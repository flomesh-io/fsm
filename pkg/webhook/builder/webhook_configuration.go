package builder

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	namerv2 "k8s.io/gengo/v2/namer"
	"k8s.io/gengo/v2/types"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/flomesh-io/fsm/pkg/constants"
)

type WebhookConfigurationBuilder struct {
	mgr                     manager.Manager
	apiType                 runtime.Object
	category                string
	webhookServiceNamespace string
	webhookServiceName      string
	caBundle                []byte
	namespaceSelector       *metav1.LabelSelector
	objectSelector          *metav1.LabelSelector
	failurePolicy           *admissionregv1.FailurePolicyType
	matchPolicy             *admissionregv1.MatchPolicyType
	sideEffects             *admissionregv1.SideEffectClass
	noMutatingWebhook       bool
	noValidatingWebhook     bool
	gvk                     schema.GroupVersionKind
	err                     error
}

func WebhookConfigurationManagedBy(m manager.Manager) *WebhookConfigurationBuilder {
	return &WebhookConfigurationBuilder{mgr: m}
}

func (b *WebhookConfigurationBuilder) For(apiType runtime.Object) *WebhookConfigurationBuilder {
	if b.apiType != nil {
		b.err = errors.New("For(...) should only be called once, could not assign multiple objects for webhook registration")
	}

	b.apiType = apiType

	gvk, err := apiutil.GVKForObject(apiType, b.mgr.GetScheme())
	if err != nil {
		b.err = err
	}

	b.gvk = gvk

	return b
}

func (b *WebhookConfigurationBuilder) WithWebhookServiceNamespace(webhookServiceNamespace string) *WebhookConfigurationBuilder {
	b.webhookServiceNamespace = webhookServiceNamespace
	return b
}

func (b *WebhookConfigurationBuilder) WithWebhookServiceName(webhookServiceName string) *WebhookConfigurationBuilder {
	b.webhookServiceName = webhookServiceName
	return b
}

func (b *WebhookConfigurationBuilder) WithCABundle(caBundle []byte) *WebhookConfigurationBuilder {
	b.caBundle = caBundle
	return b
}

func (b *WebhookConfigurationBuilder) WithCategory(category string) *WebhookConfigurationBuilder {
	b.category = category
	return b
}

func (b *WebhookConfigurationBuilder) WithFailurePolicy(failurePolicy *admissionregv1.FailurePolicyType) *WebhookConfigurationBuilder {
	b.failurePolicy = failurePolicy
	return b
}

func (b *WebhookConfigurationBuilder) WithMatchPolicy(matchPolicy *admissionregv1.MatchPolicyType) *WebhookConfigurationBuilder {
	b.matchPolicy = matchPolicy
	return b
}

func (b *WebhookConfigurationBuilder) WithSideEffects(sideEffects *admissionregv1.SideEffectClass) *WebhookConfigurationBuilder {
	b.sideEffects = sideEffects
	return b
}

func (b *WebhookConfigurationBuilder) WithNamespaceSelector(namespaceSelector *metav1.LabelSelector) *WebhookConfigurationBuilder {
	b.namespaceSelector = namespaceSelector
	return b
}

func (b *WebhookConfigurationBuilder) WithObjectSelector(objectSelector *metav1.LabelSelector) *WebhookConfigurationBuilder {
	b.objectSelector = objectSelector
	return b
}

func (b *WebhookConfigurationBuilder) WithoutMutatingWebhook() *WebhookConfigurationBuilder {
	b.noMutatingWebhook = true
	return b
}

func (b *WebhookConfigurationBuilder) WithoutValidatingWebhook() *WebhookConfigurationBuilder {
	b.noValidatingWebhook = true
	return b
}

func (b *WebhookConfigurationBuilder) Complete() (*WebhookConfigurationBuilder, error) {
	if b.webhookServiceNamespace == "" {
		return nil, fmt.Errorf("webhookServiceNamespace is empty")
	}

	if b.webhookServiceName == "" {
		return nil, fmt.Errorf("webhookServiceName is empty")
	}

	if len(b.caBundle) == 0 {
		return nil, fmt.Errorf("caBundle is empty")
	}

	if b.err != nil {
		return nil, b.err
	}

	if b.failurePolicy == nil {
		b.failurePolicy = ptr.To(admissionregv1.Ignore)
	}

	if b.matchPolicy == nil {
		b.matchPolicy = ptr.To(admissionregv1.Exact)
	}

	if b.sideEffects == nil {
		b.sideEffects = ptr.To(admissionregv1.SideEffectClassNone)
	}

	return b, nil
}

func (b *WebhookConfigurationBuilder) GetCategory() string {
	return b.category
}

func (b *WebhookConfigurationBuilder) MutatingWebhook() *admissionregv1.MutatingWebhook {
	if b.noMutatingWebhook {
		return nil
	}

	result := &admissionregv1.MutatingWebhook{
		Name: generateMutatingWebhookName(b.gvk, b.category),
		ClientConfig: admissionregv1.WebhookClientConfig{
			Service: &admissionregv1.ServiceReference{
				Namespace: b.webhookServiceNamespace,
				Name:      b.webhookServiceName,
				Path:      ptr.To(generateMutatePath(b.gvk, b.category)),
				Port:      pointer.Int32(constants.FSMWebhookPort),
			},
			CABundle: b.caBundle,
		},
		FailurePolicy:           b.failurePolicy,
		MatchPolicy:             b.matchPolicy,
		Rules:                   b.newRules(b.gvk),
		SideEffects:             b.sideEffects,
		AdmissionReviewVersions: []string{"v1"},
	}

	if b.namespaceSelector != nil {
		result.NamespaceSelector = b.namespaceSelector
	}

	if b.objectSelector != nil {
		result.ObjectSelector = b.objectSelector
	}

	return result
}

func (b *WebhookConfigurationBuilder) ValidatingWebhook() *admissionregv1.ValidatingWebhook {
	if b.noValidatingWebhook {
		return nil
	}

	result := &admissionregv1.ValidatingWebhook{
		Name: generateValidatingWebhookName(b.gvk, b.category),
		ClientConfig: admissionregv1.WebhookClientConfig{
			Service: &admissionregv1.ServiceReference{
				Namespace: b.webhookServiceNamespace,
				Name:      b.webhookServiceName,
				Path:      ptr.To(generateValidatePath(b.gvk, b.category)),
				Port:      pointer.Int32(constants.FSMWebhookPort),
			},
			CABundle: b.caBundle,
		},
		FailurePolicy:           b.failurePolicy,
		MatchPolicy:             b.matchPolicy,
		Rules:                   b.newRules(b.gvk),
		SideEffects:             b.sideEffects,
		AdmissionReviewVersions: []string{"v1"},
	}

	if b.namespaceSelector != nil {
		result.NamespaceSelector = b.namespaceSelector
	}

	if b.objectSelector != nil {
		result.ObjectSelector = b.objectSelector
	}

	return result
}

func (b *WebhookConfigurationBuilder) newRules(gvk schema.GroupVersionKind) []admissionregv1.RuleWithOperations {
	return []admissionregv1.RuleWithOperations{
		{
			Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
			Rule: admissionregv1.Rule{
				APIGroups:   []string{gvk.Group},
				APIVersions: []string{gvk.Version},
				Resources:   []string{pluralName(gvk)},
			},
		},
	}
}

func generateMutatePath(gvk schema.GroupVersionKind, category string) string {
	return fmt.Sprintf("/mutate-%s-%s-%s%s", strings.ReplaceAll(gvk.Group, ".", "-"), gvk.Version, strings.ToLower(category), strings.ToLower(gvk.Kind))
}

func generateValidatePath(gvk schema.GroupVersionKind, category string) string {
	return fmt.Sprintf("/validate-%s-%s-%s%s", strings.ReplaceAll(gvk.Group, ".", "-"), gvk.Version, strings.ToLower(category), strings.ToLower(gvk.Kind))
}

func generateMutatingWebhookName(gvk schema.GroupVersionKind, category string) string {
	return fmt.Sprintf("m%s%s.%s.kb.flomesh.io", strings.ToLower(category), strings.ToLower(gvk.Kind), gvk.Version)
}

func generateValidatingWebhookName(gvk schema.GroupVersionKind, category string) string {
	return fmt.Sprintf("v%s%s.%s.kb.flomesh.io", strings.ToLower(category), strings.ToLower(gvk.Kind), gvk.Version)
}

func pluralName(gvk schema.GroupVersionKind) string {
	namer := namerv2.NewAllLowercasePluralNamer(make(map[string]string))
	t := &types.Type{Name: types.Name{Name: gvk.Kind}}
	return namer.Name(t)
}

const webhookPathStringValidation = `^((/[a-zA-Z0-9-_]+)+|/)$`

var validWebhookPathRegex = regexp.MustCompile(webhookPathStringValidation)

func generateCustomPath(customPath string) (string, error) {
	if !validWebhookPathRegex.MatchString(customPath) {
		return "", errors.New("customPath \"" + customPath + "\" does not match this regex: " + webhookPathStringValidation)
	}
	return customPath, nil
}
