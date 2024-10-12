// Package main implements FSM CLI commands and utility routines required by the CLI.
package main

import (
	"fmt"
	"os"

	cmdanalyze "sigs.k8s.io/gwctl/cmd/analyze"
	cmdapply "sigs.k8s.io/gwctl/cmd/apply"
	cmddelete "sigs.k8s.io/gwctl/cmd/delete"
	cmdget "sigs.k8s.io/gwctl/cmd/get"

	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/flomesh-io/fsm/pkg/cli"
)

var globalUsage = `The fsm cli enables you to install and manage the
Flomesh Service Mesh (FSM) in your Kubernetes cluster

To install and configure FSM, run:

   $ fsm install
`

var settings = cli.New()

func newRootCmd(config *action.Configuration, ioStreams genericiooptions.IOStreams, args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "fsm",
		Short:        "Install and manage Flomesh Service Mesh",
		Long:         globalUsage,
		SilenceUsage: true,
	}

	//cmd.SetArgs(args)

	//cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	flags := cmd.PersistentFlags()
	settings.AddFlags(flags)

	factory := settings.Factory()
	stdin := ioStreams.In
	stdout := ioStreams.Out
	stderr := ioStreams.ErrOut

	// Add subcommands here
	cmd.AddCommand(
		newMeshCmd(config, stdin, stdout),
		newEnvCmd(stdout, stderr),
		newNamespaceCmd(stdout),
		newMetricsCmd(stdout),
		newVersionCmd(stdout),
		newProxyCmd(config, factory, stdout),
		newPolicyCmd(stdout, stderr),
		newSupportCmd(config, stdout, stderr),
		newUninstallCmd(config, stdin, stdout),
		newIngressCmd(config, stdout),
		newGatewayCmd(stdout),
		newServiceLBCmd(stdout),
		newFLBCmd(config, stdout),
		newEgressGatewayCmd(config, stdout),
		cmdapply.NewCmd(factory, ioStreams),
		cmdget.NewCmd(factory, ioStreams, false),
		cmdget.NewCmd(factory, ioStreams, true),
		cmddelete.NewCmd(factory, ioStreams),
		cmdanalyze.NewCmd(factory, ioStreams),
	)

	// Add subcommands related to unmanaged environments
	if !settings.IsManaged() {
		cmd.AddCommand(
			newInstallCmd(config, stdout),
			newDashboardCmd(config, stdout),
		)
	}

	err := cmd.PersistentFlags().Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse flags: %v", err)
		os.Exit(1)
	}
	if settings.Verbose() {
		cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
			fmt.Fprintf(os.Stderr, "flag=%s, value=%s\n", flag.Name, flag.Value)
		})
	}

	return cmd
}

func initCommands() *cobra.Command {
	actionConfig := new(action.Configuration)
	cmd := newRootCmd(actionConfig, settings.IOStreams(), os.Args)
	_ = actionConfig.Init(settings.RESTClientGetter(), settings.FsmNamespace(), "secret", debug)

	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.FsmNamespace(), "secret", debug); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize action configuration: %v", err)
			os.Exit(1)
		}
	})

	return cmd
}

func main() {
	cmd := initCommands()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func debug(format string, v ...interface{}) {
	if settings.Verbose() {
		format = fmt.Sprintf("[debug] %s\n", format)
		fmt.Printf(format, v...)
	}
}
