// Package main implements fsm cni plugin.
package main

import (
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/flomesh-io/fsm/pkg/cni/plugin"
	"github.com/flomesh-io/fsm/pkg/logger"
)

func init() {
	_ = logger.SetLogLevel("warn")
}

func main() {
	skel.PluginMain(plugin.CmdAdd, plugin.CmdCheck, plugin.CmdDelete, version.All,
		fmt.Sprintf("CNI plugin fsm-cni %v", "0.1.0"))
}
