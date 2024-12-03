package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"

	"sigs.k8s.io/gwctl/pkg/common"
)

const proxyCmdDescription = `
This command consists of subcommands related to the operations
of the sidecar proxy on pods.
`

func newProxyCmd(config *action.Configuration, factory common.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "sidecar proxy operations",
		Long:  proxyCmdDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newProxyGetCmd(config, factory, out))

	return cmd
}
