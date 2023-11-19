package reconciler

import (
	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/gateway"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	mrepo "github.com/flomesh-io/fsm/pkg/manager/repo"
	"github.com/flomesh-io/fsm/pkg/version"
)

func RegisterControllers(ctx *fctx.ControllerContext) error {
	if ctx.Config.IsIngressEnabled() {
		ingressController := pipy.NewIngressController(ctx.InformerCollection, ctx.KubeClient, ctx.Broker, ctx.Config, ctx.CertificateManager)
		if err := ctx.Manager.Add(ingressController); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Ingress Controller to manager")

			return err
		}
	}

	if ctx.Config.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(ctx.KubeClient) {
		gatewayAPIClient, err := gatewayApiClientset.NewForConfig(ctx.KubeConfig)
		if err != nil {
			return err
		}

		gatewayController := gateway.NewGatewayAPIController(ctx.InformerCollection, ctx.KubeClient, gatewayAPIClient, ctx.Broker, ctx.Config, ctx.MeshName, ctx.FSMVersion)
		ctx.EventHandler = gatewayController
		if err := ctx.Manager.Add(gatewayController); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Gateway Controller to manager")
			return err
		}
	}

	if ctx.Config.IsIngressEnabled() || (ctx.Config.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(ctx.KubeClient)) {
		rebuilder := mrepo.NewRebuilder(ctx.RepoClient, ctx.Manager.GetClient(), ctx.Config)
		if err := ctx.Manager.Add(rebuilder); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Repo Rebuilder to manager")
			return err
		}
	}

	return nil
}
