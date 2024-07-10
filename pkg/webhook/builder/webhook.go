package builder

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/flomesh-io/fsm/pkg/webhook/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"
)

// WebhookBuilder builds a Webhook.
type WebhookBuilder struct {
	apiType         runtime.Object
	category        string
	customDefaulter admission.CustomDefaulter
	customValidator admission.CustomValidator
	gvk             schema.GroupVersionKind
	mgr             manager.Manager
	config          *rest.Config
	recoverPanic    bool
	logConstructor  func(base logr.Logger, req *admission.Request) logr.Logger
	err             error
}

// WebhookManagedBy returns a new webhook builder.
func WebhookManagedBy(m manager.Manager) *WebhookBuilder {
	return &WebhookBuilder{mgr: m}
}

// For takes a runtime.Object which should be a CR.
// If the given object implements the admission.Defaulter interface, a MutatingWebhook will be wired for this type.
// If the given object implements the admission.Validator interface, a ValidatingWebhook will be wired for this type.
func (blder *WebhookBuilder) For(apiType runtime.Object) *WebhookBuilder {
	if blder.apiType != nil {
		blder.err = errors.New("For(...) should only be called once, could not assign multiple objects for webhook registration")
	}
	blder.apiType = apiType
	return blder
}

// WithDefaulter takes an admission.CustomDefaulter interface, a MutatingWebhook will be wired for this type.
func (blder *WebhookBuilder) WithDefaulter(defaulter admission.CustomDefaulter) *WebhookBuilder {
	blder.customDefaulter = defaulter
	return blder
}

// WithValidator takes a admission.CustomValidator interface, a ValidatingWebhook will be wired for this type.
func (blder *WebhookBuilder) WithValidator(validator admission.CustomValidator) *WebhookBuilder {
	blder.customValidator = validator
	return blder
}

func (blder *WebhookBuilder) WithCategoryProvider(provider types.CategoryProvider) *WebhookBuilder {
	blder.category = provider.GetCategory()
	return blder
}

func (blder *WebhookBuilder) WithCategory(category string) *WebhookBuilder {
	blder.category = category
	return blder
}

// WithLogConstructor overrides the webhook's LogConstructor.
func (blder *WebhookBuilder) WithLogConstructor(logConstructor func(base logr.Logger, req *admission.Request) logr.Logger) *WebhookBuilder {
	blder.logConstructor = logConstructor
	return blder
}

// RecoverPanic indicates whether panics caused by the webhook should be recovered.
func (blder *WebhookBuilder) RecoverPanic() *WebhookBuilder {
	blder.recoverPanic = true
	return blder
}

// Complete builds the webhook.
func (blder *WebhookBuilder) Complete() error {
	// Set the Config
	blder.loadRestConfig()

	// Configure the default LogConstructor
	blder.setLogConstructor()

	// Set the Webhook if needed
	return blder.registerWebhooks()
}

func (blder *WebhookBuilder) loadRestConfig() {
	if blder.config == nil {
		blder.config = blder.mgr.GetConfig()
	}
}

func (blder *WebhookBuilder) setLogConstructor() {
	if blder.logConstructor == nil {
		blder.logConstructor = func(base logr.Logger, req *admission.Request) logr.Logger {
			log := base.WithValues(
				"webhookGroup", blder.gvk.Group,
				"webhookKind", blder.gvk.Kind,
			)
			if req != nil {
				return log.WithValues(
					blder.gvk.Kind, klog.KRef(req.Namespace, req.Name),
					"namespace", req.Namespace, "name", req.Name,
					"resource", req.Resource, "user", req.UserInfo.Username,
					"requestID", req.UID,
				)
			}
			return log
		}
	}
}

func (blder *WebhookBuilder) registerWebhooks() error {
	typ, err := blder.getType()
	if err != nil {
		return err
	}

	blder.gvk, err = apiutil.GVKForObject(typ, blder.mgr.GetScheme())
	if err != nil {
		return err
	}

	// Register webhook(s) for type
	blder.registerDefaultingWebhook()
	blder.registerValidatingWebhook()

	err = blder.registerConversionWebhook()
	if err != nil {
		return err
	}
	return blder.err
}

// registerDefaultingWebhook registers a defaulting webhook if necessary.
func (blder *WebhookBuilder) registerDefaultingWebhook() {
	mwh := blder.getDefaultingWebhook()
	if mwh != nil {
		mwh.LogConstructor = blder.logConstructor
		path := generateMutatePath(blder.gvk, blder.category)

		// Checking if the path is already registered.
		// If so, just skip it.
		if !blder.isAlreadyHandled(path) {
			log.Info().Msgf("Registering a mutating webhook, GVK %q, path: %s", blder.gvk, path)
			blder.mgr.GetWebhookServer().Register(path, mwh)
		}
	}
}

func (blder *WebhookBuilder) getDefaultingWebhook() *admission.Webhook {
	if defaulter := blder.customDefaulter; defaulter != nil {
		return admission.WithCustomDefaulter(blder.mgr.GetScheme(), blder.apiType, defaulter).WithRecoverPanic(blder.recoverPanic)
	}
	if defaulter, ok := blder.apiType.(admission.Defaulter); ok {
		return admission.DefaultingWebhookFor(blder.mgr.GetScheme(), defaulter).WithRecoverPanic(blder.recoverPanic)
	}
	log.Info().Msgf(
		"skip registering a mutating webhook, object does not implement admission.Defaulter or WithDefaulter wasn't called, GVK: %q", blder.gvk)
	return nil
}

// registerValidatingWebhook registers a validating webhook if necessary.
func (blder *WebhookBuilder) registerValidatingWebhook() {
	vwh := blder.getValidatingWebhook()
	if vwh != nil {
		vwh.LogConstructor = blder.logConstructor
		path := generateValidatePath(blder.gvk, blder.category)

		// Checking if the path is already registered.
		// If so, just skip it.
		if !blder.isAlreadyHandled(path) {
			log.Info().Msgf("Registering a validating webhook, GVK: %q, path: %s", blder.gvk, path)
			blder.mgr.GetWebhookServer().Register(path, vwh)
		}
	}
}

func (blder *WebhookBuilder) getValidatingWebhook() *admission.Webhook {
	if validator := blder.customValidator; validator != nil {
		return admission.WithCustomValidator(blder.mgr.GetScheme(), blder.apiType, validator).WithRecoverPanic(blder.recoverPanic)
	}
	if validator, ok := blder.apiType.(admission.Validator); ok {
		return admission.ValidatingWebhookFor(blder.mgr.GetScheme(), validator).WithRecoverPanic(blder.recoverPanic)
	}
	log.Info().Msgf(
		"skip registering a validating webhook, object does not implement admission.Validator or WithValidator wasn't called, GVK: %q", blder.gvk)
	return nil
}

func (blder *WebhookBuilder) registerConversionWebhook() error {
	ok, err := conversion.IsConvertible(blder.mgr.GetScheme(), blder.apiType)
	if err != nil {
		log.Error().Msgf("Conversion check failed, GVK %q: %s", blder.gvk, err)
		return err
	}
	if ok {
		if !blder.isAlreadyHandled("/convert") {
			blder.mgr.GetWebhookServer().Register("/convert", conversion.NewWebhookHandler(blder.mgr.GetScheme()))
		}
		log.Info().Msgf("Conversion webhook enabled, GVK: %q", blder.gvk)
	}

	return nil
}

func (blder *WebhookBuilder) getType() (runtime.Object, error) {
	if blder.apiType != nil {
		return blder.apiType, nil
	}
	return nil, errors.New("For() must be called with a valid object")
}

func (blder *WebhookBuilder) isAlreadyHandled(path string) bool {
	if blder.mgr.GetWebhookServer().WebhookMux() == nil {
		return false
	}
	h, p := blder.mgr.GetWebhookServer().WebhookMux().Handler(&http.Request{URL: &url.URL{Path: path}})
	if p == path && h != nil {
		return true
	}
	return false
}
