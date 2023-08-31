package main

import (
	"io"

	"helm.sh/helm/v3/pkg/action"

	"github.com/spf13/cobra"
)

const flbDescription = `
This command consists of multiple subcommands related to managing flb controller
associated with fsm installations.
`

var (
	flbManifestFiles = []string{
		"templates/fsm-flb-secret.yaml",
	}
)

func newFLBCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "flb",
		Short:   "manage fsm FLB",
		Aliases: []string{"flb"},
		Long:    flbDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newFLBEnableCmd(config, out))
	cmd.AddCommand(newFLBDisableCmd(out))

	return cmd
}
