package main

import (
	"io"

	"helm.sh/helm/v3/pkg/action"

	"github.com/spf13/cobra"
)

const connectorDescription = `
This command consists of multiple subcommands related to managing connector
associated with fsm installations.
`

func newConnectorCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connector",
		Short:   "manage fsm connector",
		Aliases: []string{"con"},
		Long:    connectorDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newConnectorEnable(config, out))
	cmd.AddCommand(newConnectorDisable(out))
	return cmd
}
