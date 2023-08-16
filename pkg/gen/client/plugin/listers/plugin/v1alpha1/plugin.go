/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/plugin/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PluginLister helps list Plugins.
// All objects returned here must be treated as read-only.
type PluginLister interface {
	// List lists all Plugins in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Plugin, err error)
	// Get retrieves the Plugin from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Plugin, error)
	PluginListerExpansion
}

// pluginLister implements the PluginLister interface.
type pluginLister struct {
	indexer cache.Indexer
}

// NewPluginLister returns a new PluginLister.
func NewPluginLister(indexer cache.Indexer) PluginLister {
	return &pluginLister{indexer: indexer}
}

// List lists all Plugins in the indexer.
func (s *pluginLister) List(selector labels.Selector) (ret []*v1alpha1.Plugin, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Plugin))
	})
	return ret, err
}

// Get retrieves the Plugin from the index for a given name.
func (s *pluginLister) Get(name string) (*v1alpha1.Plugin, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("plugin"), name)
	}
	return obj.(*v1alpha1.Plugin), nil
}
