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

func newIngressCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "manage fsm ingress",
		Aliases: []string{"ing"},
		Long:    ingressDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newIngressEnable(config, out))
	//cmd.AddCommand(newIngressDisable(out))

	return cmd
}
