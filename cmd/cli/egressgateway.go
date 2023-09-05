package main

import (
	"io"

	"helm.sh/helm/v3/pkg/action"

	"github.com/spf13/cobra"
)

const egressGatewayDescription = `
This command consists of multiple subcommands related to managing egress gateway
associated with fsm installations.
`

var (
	egressGatewayManifestFiles = []string{
		"templates/egress-gateway-configmap.yaml",
		"templates/egress-gateway-deployment.yaml",
		"templates/egress-gateway-service.yaml",
	}
)

func newEgressGatewayCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "egressgateway",
		Short:   "manage fsm egress-gateway",
		Aliases: []string{"egw"},
		Long:    egressGatewayDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newEgressGatewayEnable(config, out))
	cmd.AddCommand(newEgressGatewayDisable(out))

	return cmd
}
