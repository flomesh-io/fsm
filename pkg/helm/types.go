// Package helm provides utilities for helm
package helm

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("helm-utilities")
)

type YAMLHandlerFunc func(dynamicClient dynamic.Interface, mapper meta.RESTMapper, manifest string) error
