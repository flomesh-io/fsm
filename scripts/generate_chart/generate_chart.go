// Package main implements generate chart application.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flomesh-io/fsm/pkg/cli"
)

func main() {
	// Path relative to the Makefile where this is invoked.
	chartPath := filepath.Join("charts", "fsm")
	source, err := cli.GetChartSource(chartPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error getting chart source:", err)
		os.Exit(1)
	}
	fmt.Print(string(source))
}
