package kube

import (
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/logger"
)

const (
	// providerName is the name of the Kubernetes client that implements service.Provider and endpoint.Provider interfaces
	providerName = "kubernetes"
)

var (
	log = logger.New("kube-provider")
)

// client is the type used to represent the k8s client for endpoints and service provider
type client struct {
	kubeController   k8s.Controller
	meshConfigurator configurator.Configurator
}
