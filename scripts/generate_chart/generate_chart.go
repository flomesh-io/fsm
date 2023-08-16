// Package main implements generate chart application.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/flomesh-io/fsm/pkg/cli"
)

var (
	chartName string
)

var (
	flags = pflag.NewFlagSet(`generate_chart`, pflag.ExitOnError)
)

func init() {
	flags.StringVar(&chartName, "chart-name", "", "Chart name")
}

func main() {
	if err := parseFlags(); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing cmd line arguments:", err)
		os.Exit(1)
	}

	// Path relative to the Makefile where this is invoked.
	chartPath := filepath.Join("charts", chartName)
	source, err := cli.GetChartSource(chartPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error getting chart source:", err)
		os.Exit(1)
	}
	fmt.Print(string(source))
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}
