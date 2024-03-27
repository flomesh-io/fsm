package reconciler

import (
	"context"

	"github.com/flomesh-io/fsm/pkg/gateway/status"
	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/gateway"
	"github.com/flomesh-io/fsm/pkg/ingress/providers/pipy"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	mrepo "github.com/flomesh-io/fsm/pkg/manager/repo"
	"github.com/flomesh-io/fsm/pkg/version"
)

func RegisterControllers(ctx context.Context) error {
	cctx, err := fctx.ToControllerContext(ctx)
	if err != nil {
		return err
	}

	mc := cctx.Configurator
	mgr := cctx.Manager
	kubeClient := cctx.KubeClient

	if mc.IsIngressEnabled() {
		ingressController := pipy.NewIngressController(cctx.InformerCollection, kubeClient, cctx.MsgBroker, mc, cctx.CertManager)
		if err := mgr.Add(ingressController); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Ingress Controller to manager")

			return err
		}
	}

	if mc.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(kubeClient) {
		statusHandler := status.NewUpdateHandler(log, cctx.Client)
		cctx.StatusUpdater = statusHandler.Writer()
		if err := mgr.Add(statusHandler); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Gateway Controller to manager")
			return err
		}

		gatewayAPIClient, err := gatewayApiClientset.NewForConfig(cctx.KubeConfig)
		if err != nil {
			return err
		}

		gatewayController := gateway.NewGatewayAPIController(cctx.InformerCollection, kubeClient, gatewayAPIClient, cctx.MsgBroker, mc, cctx.MeshName, cctx.FSMVersion)
		cctx.GatewayEventHandler = gatewayController
		if err := mgr.Add(gatewayController); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Gateway Controller to manager")
			return err
		}
	}

	if mc.IsIngressEnabled() || (mc.IsGatewayAPIEnabled() && version.IsSupportedK8sVersionForGatewayAPI(kubeClient)) {
		rebuilder := mrepo.NewRebuilder(cctx.RepoClient, cctx.Client, mc)
		if err := mgr.Add(rebuilder); err != nil {
			events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error add Repo Rebuilder to manager")
			return err
		}
	}

	return nil
}
