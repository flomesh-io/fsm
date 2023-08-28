package main

import (
	"io"

	"helm.sh/helm/v3/pkg/action"

	"github.com/spf13/cobra"
)

const ingressDescription = `
This command consists of multiple subcommands related to managing ingress controller
associated with fsm installations.
`

const (
	presetMeshConfigName    = "preset-mesh-config"
	presetMeshConfigJSONKey = "preset-mesh-config.json"
)

var (
	ingressManifestFiles = []string{
		"templates/fsm-ingress-class.yaml",
		"templates/fsm-ingress-deployment.yaml",
		"templates/fsm-ingress-service.yaml",
	}
)

func newIngressCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "manage fsm ingress",
		Aliases: []string{"ing"},
		Long:    ingressDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newIngressEnable(config, out))
	cmd.AddCommand(newIngressDisable(config, out))

	return cmd
}
