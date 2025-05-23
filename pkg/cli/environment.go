/*
Package cli describes the operating environment for the FSM cli and
includes convenience functions for the FSM cli.

Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

# Copyright 2020 The FSM contributors

Licensed under the MIT License
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

This package is heavily inspired by how the Helm project handles
environment variables: https://github.com/helm/helm/blob/master/pkg/cli/environment.go
*/
package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"sigs.k8s.io/gwctl/pkg/common"
)

const (
	fsmConfigEnvVar = "FSM_CONFIG"
)

const (
	installKindManaged    = "managed"
	installKindSelfHosted = "self-hosted"
)

const (
	defaultFSMNamespace = "fsm-system"
)

// EnvConfig represents the environment configuration of FSM
type EnvConfig struct {
	Install EnvConfigInstall `yaml:"install"`
}

// EnvConfigInstall represents the environment configuration of FSM install
type EnvConfigInstall struct {
	Kind         string `yaml:"kind"`
	Distribution string `yaml:"distribution"`
	Namespace    string `yaml:"namespace"`
}

// EnvSettings describes all of the cli environment settings
type EnvSettings struct {
	envConfig EnvConfig
	config    *genericclioptions.ConfigFlags
	verbose   bool
	ioStreams genericiooptions.IOStreams
	factory   common.Factory
}

// New relevant environment variables set and returns EnvSettings
func New() *EnvSettings {
	envConfig, err := envFromConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing environment configuration: %s\n", err)
		os.Exit(1)
	}

	ioStreams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	env := &EnvSettings{
		envConfig: *envConfig,
		ioStreams: ioStreams,
	}

	// bind to kubernetes config flags
	//env.config = &genericclioptions.ConfigFlags{
	//	Namespace: &env.envConfig.Install.Namespace,
	//}
	globalConfig := genericclioptions.NewConfigFlags(true).
		WithDiscoveryBurst(300).
		WithDiscoveryQPS(50.0).
		WithWarningPrinter(ioStreams)
	//globalConfig.Namespace = &env.envConfig.Install.Namespace
	env.factory = common.NewFactory(globalConfig)
	env.config = globalConfig

	return env
}

// AddFlags binds flags to the given flagset.
func (s *EnvSettings) AddFlags(fs *pflag.FlagSet) {
	s.config.AddFlags(fs)

	fs.StringVarP(&s.envConfig.Install.Namespace, "fsm-namespace", "N", s.envConfig.Install.Namespace, "namespace for fsm control plane")
	fs.BoolVar(&s.verbose, "verbose", s.verbose, "enable verbose output")

	fs.PrintDefaults()
}

// Config returns the environment config
func (s *EnvSettings) Config() EnvConfig {
	return s.envConfig
}

// RESTClientGetter gets the kubeconfig from EnvSettings
func (s *EnvSettings) RESTClientGetter() genericclioptions.RESTClientGetter {
	return s.config
}

// FsmNamespace gets the fsm-namespace from the configuration
func (s *EnvSettings) FsmNamespace() string {
	return s.envConfig.Install.Namespace
}

// Verbose gets whether verbose output is enabled from the configuration
func (s *EnvSettings) Verbose() bool {
	return s.verbose
}

// IOStreams returns the IOStreams from the configuration
func (s *EnvSettings) IOStreams() genericiooptions.IOStreams {
	return s.ioStreams
}

// Factory returns the factory from the configuration
func (s *EnvSettings) Factory() common.Factory {
	return s.factory
}

// IsManaged returns true in a managed FSM environment (ex. managed by a cloud distributor)
func (s *EnvSettings) IsManaged() bool {
	return s.envConfig.Install.Kind == installKindManaged
}

// envFromConfig returns the environment information from the config file.
// The config file is looked up as follows:
// 1. Look for config file specified in FSM_CONFIG env var. If set, use it.
// 2. If 1. is not applicable, look for a file in $HOME/.fsm/config; if file exists use it
// 3. If neither of 1. and 2. apply, use system defaults.
func envFromConfig() (*EnvConfig, error) {
	configFile, ok := os.LookupEnv(fsmConfigEnvVar)
	if !ok {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configFile = path.Join(homeDir, ".fsm", "config.yaml")
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Config file does not exist, use defaults
		return &EnvConfig{
			Install: EnvConfigInstall{
				Kind:      installKindSelfHosted,
				Namespace: defaultFSMNamespace,
			},
		}, nil
	}

	// Populate environment info from config file
	f, err := os.Open(filepath.Clean(configFile))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing file: %s\n", err)
		}
	}()

	var cfg EnvConfig
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
