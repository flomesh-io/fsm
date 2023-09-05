package main

import (
	"io"

	"github.com/spf13/cobra"
)

const gatewayDescription = `
This command consists of multiple subcommands related to managing gateway controller
associated with fsm installations.
`

func newGatewayCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gateway",
		Short:   "manage fsm gateway",
		Aliases: []string{"gw"},
		Long:    gatewayDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newGatewayEnable(out))
	cmd.AddCommand(newGatewayDisable(out))

	return cmd
}
