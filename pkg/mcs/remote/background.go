package remote

import (
	"context"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	mcscfg "github.com/flomesh-io/fsm/pkg/mcs/config"
	cctx "github.com/flomesh-io/fsm/pkg/mcs/context"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/utils"
	"k8s.io/client-go/rest"
)

func NewBackground(cluster *mcsv1alpha1.Cluster, kubeconfig *rest.Config, mc configurator.Configurator, broker *messaging.Broker) (*Background, error) {
	connCfg, err := connectorConfig(cluster, mc)
	if err != nil {
		return nil, err
	}

	background := &cctx.ConnectorContext{
		ClusterKey:      cluster.Key(),
		KubeConfig:      kubeconfig,
		ConnectorConfig: connCfg,
		Hash:            clusterHash(cluster),
	}
	_, cancel := context.WithCancel(background)
	stop := utils.RegisterExitHandlers(cancel)
	background.Cancel = cancel
	background.StopCh = stop

	connector, err := NewConnector(background, broker)
	if err != nil {
		log.Error().Msgf("Failed to create connector for cluster %q: %s", cluster.Key(), err)
		return nil, err
	}

	return &Background{
		//isInCluster: cluster.Spec.IsInCluster,
		Context:   background,
		Connector: connector,
	}, nil
}

func clusterHash(cluster *mcsv1alpha1.Cluster) string {
	return utils.SimpleHash(
		struct {
			spec            mcsv1alpha1.ClusterSpec
			resourceVersion string
			generation      int64
			uuid            string
		}{
			spec:            cluster.Spec,
			resourceVersion: cluster.ResourceVersion,
			generation:      cluster.Generation,
			uuid:            string(cluster.UID),
		},
	)
}

func connectorConfig(cluster *mcsv1alpha1.Cluster, mc configurator.Configurator) (*mcscfg.ConnectorConfig, error) {
	return mcscfg.NewConnectorConfig(
		cluster.Spec.Region,
		cluster.Spec.Zone,
		cluster.Spec.Group,
		cluster.Name,
		cluster.Spec.GatewayHost,
		cluster.Spec.GatewayPort,
		mc.GetClusterUID(),
	)
}

func (b *Background) Run() error {
	return b.Connector.Run(b.Context.StopCh)
}
