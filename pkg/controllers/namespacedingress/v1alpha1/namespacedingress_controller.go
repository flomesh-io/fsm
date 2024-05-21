/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package v1alpha1 contains controller logic for the NamespacedIngress API v1alpha1.
package v1alpha1

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/flomesh-io/fsm/pkg/constants"

	ghodssyaml "github.com/ghodss/yaml"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/controllers"
	"github.com/flomesh-io/fsm/pkg/helm"
	"github.com/flomesh-io/fsm/pkg/logger"
	mgrutils "github.com/flomesh-io/fsm/pkg/manager/utils"
	"github.com/flomesh-io/fsm/pkg/utils"
)

var (
	//go:embed chart.tgz
	chartSource []byte
)

var (
	log = logger.New("namespacedingress-controller/v1alpha1")
)

// NamespacedIngressReconciler reconciles a NamespacedIngress object
type reconciler struct {
	recorder record.EventRecorder
	fctx     *fctx.ControllerContext
}

func (r *reconciler) NeedLeaderElection() bool {
	return true
}

// NewReconciler returns a new NamespacedIngress reconciler
func NewReconciler(ctx *fctx.ControllerContext) controllers.Reconciler {
	return &reconciler{
		recorder: ctx.Manager.GetEventRecorderFor("NamespacedIngress"),
		fctx:     ctx,
	}
}

type namespacedIngressValues struct {
	NamespacedIngress *nsigv1alpha1.NamespacedIngress `json:"nsig,omitempty"`
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the NamespacedIngress closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespacedIngress object against the actual NamespacedIngress state, and then
// perform operations to make the NamespacedIngress state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mc := r.fctx.Configurator

	log.Debug().Msgf("[NSIG] Ingress Enabled = %t, Namespaced Ingress = %t", mc.IsIngressEnabled(), mc.IsNamespacedIngressEnabled())
	if !mc.IsNamespacedIngressEnabled() {
		log.Warn().Msgf("Ingress is not enabled or Ingress mode is not Namespace, ignore processing NamespacedIngress...")
		return ctrl.Result{}, nil
	}

	nsig := &nsigv1alpha1.NamespacedIngress{}
	if err := r.fctx.Get(
		ctx,
		client.ObjectKey{Name: req.Name, Namespace: req.Namespace},
		nsig,
	); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Debug().Msgf("[NSIG] NamespacedIngress resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error().Msgf("Failed to get NamespacedIngress, %v", err)
		return ctrl.Result{}, err
	}

	ctrlResult, err := r.deriveCodebases(nsig, mc)
	if err != nil {
		return ctrlResult, err
	}

	ctrlResult, err = r.updateConfig(nsig, mc)
	if err != nil {
		return ctrlResult, err
	}

	actionConfig := helm.ActionConfig(nsig.Namespace, log.Debug().Msgf)
	templateClient := helm.TemplateClient(
		actionConfig,
		fmt.Sprintf("namespaced-ingress-%s", nsig.Namespace),
		nsig.Namespace,
		constants.KubeVersion119,
	)
	if ctrlResult, err = helm.RenderChart(templateClient, nsig, chartSource, mc, r.fctx.Client, r.fctx.Scheme, r.resolveValues); err != nil {
		return ctrlResult, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) resolveValues(object metav1.Object, mc configurator.Configurator) (map[string]interface{}, error) {
	nsig, ok := object.(*nsigv1alpha1.NamespacedIngress)
	if !ok {
		return nil, fmt.Errorf("object %v is not type of nsigv1alpha1.NamespacedIngress", object)
	}

	log.Debug().Msgf("[NSIG] Resolving Values ...")

	nsigBytes, err := ghodssyaml.Marshal(&namespacedIngressValues{NamespacedIngress: nsig})
	if err != nil {
		return nil, fmt.Errorf("convert NamespacedIngress to yaml, err = %v", err)
	}
	log.Debug().Msgf("\n\nNSIG VALUES YAML:\n\n\n%s\n\n", string(nsigBytes))
	nsigValues, err := chartutil.ReadValues(nsigBytes)
	if err != nil {
		return nil, err
	}

	finalValues := nsigValues.AsMap()

	overrides := []string{
		fmt.Sprintf("fsm.image.registry=%s", mc.GetImageRegistry()),
		fmt.Sprintf("fsm.image.tag=%s", mc.GetImageTag()),
		fmt.Sprintf("fsm.image.pullPolicy=%s", mc.GetImagePullPolicy()),
		fmt.Sprintf("fsm.fsmNamespace=%s", mc.GetFSMNamespace()),
		fmt.Sprintf("fsm.fsmIngress.logLevel=%s", mc.GetFSMIngressLogLevel()),
		fmt.Sprintf("fsm.meshName=%s", r.fctx.MeshName),
	}

	for _, ov := range overrides {
		if err := strvals.ParseInto(ov, finalValues); err != nil {
			return nil, err
		}
	}

	return finalValues, nil
}

func (r *reconciler) deriveCodebases(nsig *nsigv1alpha1.NamespacedIngress, _ configurator.Configurator) (ctrl.Result, error) {
	repoClient := r.fctx.RepoClient

	ingressPath := utils.NamespacedIngressCodebasePath(nsig.Namespace)
	parentPath := utils.IngressCodebasePath()
	if err := repoClient.DeriveCodebase(ingressPath, parentPath); err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) updateConfig(nsig *nsigv1alpha1.NamespacedIngress, mc configurator.Configurator) (ctrl.Result, error) {
	if mc.IsNamespacedIngressEnabled() {
		repoClient := r.fctx.RepoClient
		basepath := utils.NamespacedIngressCodebasePath(nsig.Namespace)

		err := mgrutils.UpdateIngressHTTPConfig(basepath, repoClient, mc, nsig)
		if err != nil {
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}

		if nsig.Spec.TLS.Enabled {
			// TLS offload
			err := mgrutils.IssueCertForIngress(basepath, repoClient, r.fctx.CertManager, mc, nsig)
			if err != nil {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
		} else {
			err := mgrutils.UpdateIngressTLSConfig(basepath, repoClient, mc, nsig)
			if err != nil {
				return ctrl.Result{RequeueAfter: 1 * time.Second}, err
			}
		}

		// SSL passthrough
		err = mgrutils.UpdateSSLPassthrough(
			basepath,
			repoClient,
			nsig.Spec.TLS.SSLPassthrough.Enabled,
			*nsig.Spec.TLS.SSLPassthrough.UpstreamPort,
		)
		if err != nil {
			return ctrl.Result{RequeueAfter: 1 * time.Second}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nsigv1alpha1.NamespacedIngress{}).
		Complete(r)
}
