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
// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=gateway.flomesh.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("accesscontrolpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().AccessControlPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("circuitbreakingpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().CircuitBreakingPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("faultinjectionpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().FaultInjectionPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("healthcheckpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().HealthCheckPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("loadbalancerpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().LoadBalancerPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("ratelimitpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().RateLimitPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("sessionstickypolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().SessionStickyPolicies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("upstreamtlspolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Gateway().V1alpha1().UpstreamTLSPolicies().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
