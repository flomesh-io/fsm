//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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
// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CircuitBreaker) DeepCopyInto(out *CircuitBreaker) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CircuitBreaker.
func (in *CircuitBreaker) DeepCopy() *CircuitBreaker {
	if in == nil {
		return nil
	}
	out := new(CircuitBreaker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CircuitBreaker) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CircuitBreakerList) DeepCopyInto(out *CircuitBreakerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CircuitBreaker, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CircuitBreakerList.
func (in *CircuitBreakerList) DeepCopy() *CircuitBreakerList {
	if in == nil {
		return nil
	}
	out := new(CircuitBreakerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CircuitBreakerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CircuitBreakerResponse) DeepCopyInto(out *CircuitBreakerResponse) {
	*out = *in
	if in.StatusCode != nil {
		in, out := &in.StatusCode, &out.StatusCode
		*out = new(int32)
		**out = **in
	}
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Body != nil {
		in, out := &in.Body, &out.Body
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CircuitBreakerResponse.
func (in *CircuitBreakerResponse) DeepCopy() *CircuitBreakerResponse {
	if in == nil {
		return nil
	}
	out := new(CircuitBreakerResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CircuitBreakerSpec) DeepCopyInto(out *CircuitBreakerSpec) {
	*out = *in
	if in.LatencyThreshold != nil {
		in, out := &in.LatencyThreshold, &out.LatencyThreshold
		*out = new(v1.Duration)
		**out = **in
	}
	if in.ErrorCountThreshold != nil {
		in, out := &in.ErrorCountThreshold, &out.ErrorCountThreshold
		*out = new(int32)
		**out = **in
	}
	if in.ErrorRatioThreshold != nil {
		in, out := &in.ErrorRatioThreshold, &out.ErrorRatioThreshold
		*out = new(float32)
		**out = **in
	}
	if in.ConcurrencyThreshold != nil {
		in, out := &in.ConcurrencyThreshold, &out.ConcurrencyThreshold
		*out = new(int32)
		**out = **in
	}
	if in.CheckInterval != nil {
		in, out := &in.CheckInterval, &out.CheckInterval
		*out = new(v1.Duration)
		**out = **in
	}
	if in.BreakInterval != nil {
		in, out := &in.BreakInterval, &out.BreakInterval
		*out = new(v1.Duration)
		**out = **in
	}
	if in.CircuitBreakerResponse != nil {
		in, out := &in.CircuitBreakerResponse, &out.CircuitBreakerResponse
		*out = new(CircuitBreakerResponse)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CircuitBreakerSpec.
func (in *CircuitBreakerSpec) DeepCopy() *CircuitBreakerSpec {
	if in == nil {
		return nil
	}
	out := new(CircuitBreakerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CircuitBreakerStatus) DeepCopyInto(out *CircuitBreakerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CircuitBreakerStatus.
func (in *CircuitBreakerStatus) DeepCopy() *CircuitBreakerStatus {
	if in == nil {
		return nil
	}
	out := new(CircuitBreakerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjection) DeepCopyInto(out *FaultInjection) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjection.
func (in *FaultInjection) DeepCopy() *FaultInjection {
	if in == nil {
		return nil
	}
	out := new(FaultInjection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FaultInjection) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjectionAbort) DeepCopyInto(out *FaultInjectionAbort) {
	*out = *in
	if in.Response != nil {
		in, out := &in.Response, &out.Response
		*out = new(FaultInjectionResponse)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjectionAbort.
func (in *FaultInjectionAbort) DeepCopy() *FaultInjectionAbort {
	if in == nil {
		return nil
	}
	out := new(FaultInjectionAbort)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjectionDelay) DeepCopyInto(out *FaultInjectionDelay) {
	*out = *in
	if in.Min != nil {
		in, out := &in.Min, &out.Min
		*out = new(v1.Duration)
		**out = **in
	}
	if in.Max != nil {
		in, out := &in.Max, &out.Max
		*out = new(v1.Duration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjectionDelay.
func (in *FaultInjectionDelay) DeepCopy() *FaultInjectionDelay {
	if in == nil {
		return nil
	}
	out := new(FaultInjectionDelay)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjectionList) DeepCopyInto(out *FaultInjectionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]FaultInjection, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjectionList.
func (in *FaultInjectionList) DeepCopy() *FaultInjectionList {
	if in == nil {
		return nil
	}
	out := new(FaultInjectionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FaultInjectionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjectionResponse) DeepCopyInto(out *FaultInjectionResponse) {
	*out = *in
	if in.StatusCode != nil {
		in, out := &in.StatusCode, &out.StatusCode
		*out = new(int32)
		**out = **in
	}
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Body != nil {
		in, out := &in.Body, &out.Body
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjectionResponse.
func (in *FaultInjectionResponse) DeepCopy() *FaultInjectionResponse {
	if in == nil {
		return nil
	}
	out := new(FaultInjectionResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjectionSpec) DeepCopyInto(out *FaultInjectionSpec) {
	*out = *in
	if in.Delay != nil {
		in, out := &in.Delay, &out.Delay
		*out = new(FaultInjectionDelay)
		(*in).DeepCopyInto(*out)
	}
	if in.Abort != nil {
		in, out := &in.Abort, &out.Abort
		*out = new(FaultInjectionAbort)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjectionSpec.
func (in *FaultInjectionSpec) DeepCopy() *FaultInjectionSpec {
	if in == nil {
		return nil
	}
	out := new(FaultInjectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FaultInjectionStatus) DeepCopyInto(out *FaultInjectionStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FaultInjectionStatus.
func (in *FaultInjectionStatus) DeepCopy() *FaultInjectionStatus {
	if in == nil {
		return nil
	}
	out := new(FaultInjectionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Filter) DeepCopyInto(out *Filter) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Filter.
func (in *Filter) DeepCopy() *Filter {
	if in == nil {
		return nil
	}
	out := new(Filter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Filter) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterDefinition) DeepCopyInto(out *FilterDefinition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterDefinition.
func (in *FilterDefinition) DeepCopy() *FilterDefinition {
	if in == nil {
		return nil
	}
	out := new(FilterDefinition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FilterDefinition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterDefinitionList) DeepCopyInto(out *FilterDefinitionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]FilterDefinition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterDefinitionList.
func (in *FilterDefinitionList) DeepCopy() *FilterDefinitionList {
	if in == nil {
		return nil
	}
	out := new(FilterDefinitionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FilterDefinitionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterDefinitionSpec) DeepCopyInto(out *FilterDefinitionSpec) {
	*out = *in
	if in.Scope != nil {
		in, out := &in.Scope, &out.Scope
		*out = new(FilterScope)
		**out = **in
	}
	if in.Protocol != nil {
		in, out := &in.Protocol, &out.Protocol
		*out = new(FilterProtocol)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterDefinitionSpec.
func (in *FilterDefinitionSpec) DeepCopy() *FilterDefinitionSpec {
	if in == nil {
		return nil
	}
	out := new(FilterDefinitionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterDefinitionStatus) DeepCopyInto(out *FilterDefinitionStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterDefinitionStatus.
func (in *FilterDefinitionStatus) DeepCopy() *FilterDefinitionStatus {
	if in == nil {
		return nil
	}
	out := new(FilterDefinitionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterList) DeepCopyInto(out *FilterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Filter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterList.
func (in *FilterList) DeepCopy() *FilterList {
	if in == nil {
		return nil
	}
	out := new(FilterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FilterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterSpec) DeepCopyInto(out *FilterSpec) {
	*out = *in
	out.DefinitionRef = in.DefinitionRef
	out.ConfigRef = in.ConfigRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterSpec.
func (in *FilterSpec) DeepCopy() *FilterSpec {
	if in == nil {
		return nil
	}
	out := new(FilterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FilterStatus) DeepCopyInto(out *FilterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FilterStatus.
func (in *FilterStatus) DeepCopy() *FilterStatus {
	if in == nil {
		return nil
	}
	out := new(FilterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListenerFilter) DeepCopyInto(out *ListenerFilter) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListenerFilter.
func (in *ListenerFilter) DeepCopy() *ListenerFilter {
	if in == nil {
		return nil
	}
	out := new(ListenerFilter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ListenerFilter) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListenerFilterList) DeepCopyInto(out *ListenerFilterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ListenerFilter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListenerFilterList.
func (in *ListenerFilterList) DeepCopy() *ListenerFilterList {
	if in == nil {
		return nil
	}
	out := new(ListenerFilterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ListenerFilterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListenerFilterSpec) DeepCopyInto(out *ListenerFilterSpec) {
	*out = *in
	if in.TargetRefs != nil {
		in, out := &in.TargetRefs, &out.TargetRefs
		*out = make([]LocalTargetReferenceWithPort, len(*in))
		copy(*out, *in)
	}
	out.DefinitionRef = in.DefinitionRef
	out.ConfigRef = in.ConfigRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListenerFilterSpec.
func (in *ListenerFilterSpec) DeepCopy() *ListenerFilterSpec {
	if in == nil {
		return nil
	}
	out := new(ListenerFilterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListenerFilterStatus) DeepCopyInto(out *ListenerFilterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListenerFilterStatus.
func (in *ListenerFilterStatus) DeepCopy() *ListenerFilterStatus {
	if in == nil {
		return nil
	}
	out := new(ListenerFilterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LocalTargetReferenceWithPort) DeepCopyInto(out *LocalTargetReferenceWithPort) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LocalTargetReferenceWithPort.
func (in *LocalTargetReferenceWithPort) DeepCopy() *LocalTargetReferenceWithPort {
	if in == nil {
		return nil
	}
	out := new(LocalTargetReferenceWithPort)
	in.DeepCopyInto(out)
	return out
}
