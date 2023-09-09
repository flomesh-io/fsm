package main

import (
	"io"

	"github.com/spf13/cobra"
)

func newNamespacedIngressCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespaced",
		Short:   "manage fsm NamespacedIngress",
		Aliases: []string{"nsig"},
		Long:    ingressDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newNamespacedIngressEnableCmd(out))
	cmd.AddCommand(newNamespacedIngressDisableCmd(out))

	return cmd
}
