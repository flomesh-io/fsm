package main

import (
	"io"

	"github.com/spf13/cobra"
)

const serviceLBDescription = `
This command consists of multiple subcommands related to managing service-lb controller
associated with fsm installations.
`

func newServiceLBCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "servicelb",
		Short:   "manage fsm service-lb",
		Aliases: []string{"slb"},
		Long:    serviceLBDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newServiceLBEnableCmd(out))
	cmd.AddCommand(newServiceLBDisableCmd(out))

	return cmd
}
