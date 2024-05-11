package flb

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flomesh-io/fsm/pkg/constants"

	"github.com/flomesh-io/fsm/pkg/utils"

	"github.com/go-resty/resty/v2"
	"github.com/sethvargo/go-retry"

	"github.com/flomesh-io/fsm/pkg/flb"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
)

// reconciler reconciles a Secret object
type secretReconciler struct {
	recorder   record.EventRecorder
	fctx       *fctx.ControllerContext
	settingMgr *SettingManager
	cache      map[types.NamespacedName]*corev1.Secret
}

func (r *secretReconciler) NeedLeaderElection() bool {
	return true
}

type CertRequest struct {
	Data CertData `json:"data"`
}

type CertData struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content Cert   `json:"content"`
}

type Cert struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type CertDeleteRequest struct {
	Name string `json:"name"`
}

// NewSecretReconciler returns a new reconciler for Secret
func NewSecretReconciler(ctx *fctx.ControllerContext, settingManager *SettingManager) controllers.Reconciler {
	log.Info().Msgf("Creating FLB secret reconciler ...")

	return &secretReconciler{
		recorder:   ctx.Manager.GetEventRecorderFor("FLB"),
		fctx:       ctx,
		settingMgr: settingManager,
		cache:      make(map[types.NamespacedName]*corev1.Secret),
	}
}

func (r *secretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	secret := &corev1.Secret{}
	if err := r.fctx.Get(
		ctx,
		req.NamespacedName,
		secret,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info().Msgf("Secret %s/%s resource not found. Ignoring since object must be deleted", req.Namespace, req.Name)

			// get sec from cache as it's not found, we don't have enough info to pop out
			sec, ok := r.cache[req.NamespacedName]
			if !ok {
				log.Warn().Msgf("Secret %s not found in cache", req.NamespacedName)
				return ctrl.Result{}, nil
			}

			if flb.IsFLBTLSSecret(sec) {
				result, err := r.deleteSecretFromFLB(ctx, sec)
				if err != nil {
					return result, err
				}

				delete(r.cache, req.NamespacedName)
				return ctrl.Result{}, nil
			}
		}

		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get Secret, %v", err)
		return ctrl.Result{}, err
	}

	if flb.IsFLBTLSSecret(secret) {
		r.cache[req.NamespacedName] = secret.DeepCopy()

		if result, err := r.settingMgr.CheckSetting(secret); err != nil {
			return result, err
		}

		if secret.DeletionTimestamp != nil {
			result, err := r.deleteSecretFromFLB(ctx, secret)
			if err != nil {
				return result, err
			}

			delete(r.cache, req.NamespacedName)
			return ctrl.Result{}, nil
		}

		return r.createOrUpdateFLBSecret(ctx, secret)
	}

	return ctrl.Result{}, nil
}

func (r *secretReconciler) createOrUpdateFLBSecret(ctx context.Context, secret *corev1.Secret) (ctrl.Result, error) {
	oldHash := getSecretHash(secret)
	hash := utils.SimpleHash(secret.Data)

	if oldHash != hash {
		data := CertRequest{
			Data: CertData{
				Name: secretKey(r.settingMgr.GetSetting(secret.Namespace), secret.Namespace, secret.Name),
				Type: "api",
				Content: Cert{
					Cert: string(secret.Data[corev1.TLSCertKey]),
					Key:  string(secret.Data[corev1.TLSPrivateKeyKey]),
				},
			},
		}

		if err := r.updateFLBSecret(secret, data, false); err != nil {
			return ctrl.Result{}, err
		}

		if len(secret.Annotations) == 0 {
			secret.Annotations = make(map[string]string)
		}

		secret.Annotations[constants.FLBHashAnnotation] = hash
		if err := r.fctx.Update(ctx, secret); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *secretReconciler) deleteSecretFromFLB(_ context.Context, secret *corev1.Secret) (ctrl.Result, error) {
	if flb.IsFLBTLSSecret(secret) {
		log.Debug().Msgf("Secret %s/%s is being deleted from FLB ...", secret.Namespace, secret.Name)

		setting := r.settingMgr.GetSetting(secret.Namespace)
		secretName := secretKey(setting, secret.Namespace, secret.Name)

		if err := r.updateFLBSecret(secret, CertDeleteRequest{Name: secretName}, true); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *secretReconciler) updateFLBSecret(secret *corev1.Secret, request interface{}, del bool) error {
	setting := r.settingMgr.GetSetting(secret.Namespace)

	if err := setting.UpdateToken(); err != nil {
		log.Error().Msgf("Login to FLB failed: %s", err)
		defer r.recorder.Eventf(secret, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", err)

		return err
	}

	if err := retry.Fibonacci(context.TODO(), 1*time.Second, func(ctx context.Context) error {
		statusCode, apiErr := r.invokeFLBAPI(secret.Namespace, request, del)

		if apiErr != nil {
			if statusCode == http.StatusUnauthorized {
				if loginErr := setting.ForceUpdateToken(); loginErr != nil {
					log.Error().Msgf("Login to FLB failed: %s", loginErr)
					defer r.recorder.Eventf(secret, corev1.EventTypeWarning, "LoginFailed", "Login to FLB failed: %s", loginErr)

					return loginErr
				}

				return retry.RetryableError(apiErr)
			}

			defer r.recorder.Eventf(secret, corev1.EventTypeWarning, "InvokeFLBApiError", "Failed to invoke FLB API: %s", apiErr)
			return apiErr
		}

		return nil
	}); err != nil {
		log.Error().Msgf("failed to update FLB: %s", err)
		defer r.recorder.Eventf(secret, corev1.EventTypeWarning, "UpdateFLBFailed", "Failed to update FLB: %s", err)

		return err
	}

	return nil
}
func (r *secretReconciler) invokeFLBAPI(namespace string, body interface{}, del bool) (int, error) {
	setting := r.settingMgr.GetSetting(namespace)
	request := setting.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader(flbUserHeaderName, setting.flbUser).
		SetHeader(flbK8sClusterHeaderName, setting.k8sCluster).
		SetAuthToken(setting.token).
		SetBody(body)

	var resp *resty.Response
	var err error
	if del {
		resp, err = request.Post(flb.DeleteCertAPIPath)
	} else {
		resp, err = request.Post(flb.CertAPIPath)
	}

	if err != nil {
		log.Error().Msgf("error happened while trying to update FLB secret, %s", err.Error())
		return -1, err
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return http.StatusUnauthorized, fmt.Errorf("invalid token")
	}

	if resp.StatusCode() != http.StatusOK {
		log.Error().Msgf("FLB server responsed with StatusCode: %d", resp.StatusCode())
		return resp.StatusCode(), fmt.Errorf("%d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return http.StatusOK, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *secretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&corev1.Secret{},
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				secret, ok := obj.(*corev1.Secret)
				if !ok {
					log.Warn().Msgf("unexpected object type: %T", obj)
					return false
				}

				return flb.IsFLBTLSSecret(secret)
			})),
		).Complete(r)
}
