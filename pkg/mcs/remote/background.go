package remote

import (
	"context"

	"k8s.io/client-go/rest"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/configurator"
	mcscfg "github.com/flomesh-io/fsm/pkg/mcs/config"
	cctx "github.com/flomesh-io/fsm/pkg/mcs/context"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// NewBackground creates a new background process for a cluster
func NewBackground(cluster *mcsv1alpha1.Cluster, kubeconfig *rest.Config, mc configurator.Configurator, broker *messaging.Broker) (*Background, error) {
	log.Debug().Msgf("Creating background process for cluster %q", cluster.Key())

	connCfg, err := connectorConfig(cluster, mc)
	if err != nil {
		return nil, err
	}

	background := &cctx.ConnectorContext{
		ClusterKey:        cluster.Key(),
		KubeConfig:        kubeconfig,
		ConnectorConfig:   connCfg,
		Hash:              clusterHash(cluster),
		FsmNamespace:      cluster.Spec.FsmNamespace,
		FsmMeshConfigName: cluster.Spec.FsmMeshConfigName,
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

// Run starts the background process
func (b *Background) Run() error {
	return b.Connector.Run(b.Context.StopCh)
}
