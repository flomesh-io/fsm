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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulAuthSpec) DeepCopyInto(out *ConsulAuthSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulAuthSpec.
func (in *ConsulAuthSpec) DeepCopy() *ConsulAuthSpec {
	if in == nil {
		return nil
	}
	out := new(ConsulAuthSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulConnector) DeepCopyInto(out *ConsulConnector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulConnector.
func (in *ConsulConnector) DeepCopy() *ConsulConnector {
	if in == nil {
		return nil
	}
	out := new(ConsulConnector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ConsulConnector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulConnectorList) DeepCopyInto(out *ConsulConnectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ConsulConnector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulConnectorList.
func (in *ConsulConnectorList) DeepCopy() *ConsulConnectorList {
	if in == nil {
		return nil
	}
	out := new(ConsulConnectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ConsulConnectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulSpec) DeepCopyInto(out *ConsulSpec) {
	*out = *in
	out.Auth = in.Auth
	in.SyncToK8S.DeepCopyInto(&out.SyncToK8S)
	in.SyncFromK8S.DeepCopyInto(&out.SyncFromK8S)
	if in.Limiter != nil {
		in, out := &in.Limiter, &out.Limiter
		*out = new(Limiter)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulSpec.
func (in *ConsulSpec) DeepCopy() *ConsulSpec {
	if in == nil {
		return nil
	}
	out := new(ConsulSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulStatus) DeepCopyInto(out *ConsulStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulStatus.
func (in *ConsulStatus) DeepCopy() *ConsulStatus {
	if in == nil {
		return nil
	}
	out := new(ConsulStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulSyncFromK8SSpec) DeepCopyInto(out *ConsulSyncFromK8SSpec) {
	*out = *in
	if in.AppendTags != nil {
		in, out := &in.AppendTags, &out.AppendTags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AppendMetadatas != nil {
		in, out := &in.AppendMetadatas, &out.AppendMetadatas
		*out = make([]Metadata, len(*in))
		copy(*out, *in)
	}
	if in.AllowK8sNamespaces != nil {
		in, out := &in.AllowK8sNamespaces, &out.AllowK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DenyK8sNamespaces != nil {
		in, out := &in.DenyK8sNamespaces, &out.DenyK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulSyncFromK8SSpec.
func (in *ConsulSyncFromK8SSpec) DeepCopy() *ConsulSyncFromK8SSpec {
	if in == nil {
		return nil
	}
	out := new(ConsulSyncFromK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConsulSyncToK8SSpec) DeepCopyInto(out *ConsulSyncToK8SSpec) {
	*out = *in
	if in.FilterMetadatas != nil {
		in, out := &in.FilterMetadatas, &out.FilterMetadatas
		*out = make([]Metadata, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConsulSyncToK8SSpec.
func (in *ConsulSyncToK8SSpec) DeepCopy() *ConsulSyncToK8SSpec {
	if in == nil {
		return nil
	}
	out := new(ConsulSyncToK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EgressSelectorSpec) DeepCopyInto(out *EgressSelectorSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EgressSelectorSpec.
func (in *EgressSelectorSpec) DeepCopy() *EgressSelectorSpec {
	if in == nil {
		return nil
	}
	out := new(EgressSelectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EurekaConnector) DeepCopyInto(out *EurekaConnector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EurekaConnector.
func (in *EurekaConnector) DeepCopy() *EurekaConnector {
	if in == nil {
		return nil
	}
	out := new(EurekaConnector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EurekaConnector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EurekaConnectorList) DeepCopyInto(out *EurekaConnectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]EurekaConnector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EurekaConnectorList.
func (in *EurekaConnectorList) DeepCopy() *EurekaConnectorList {
	if in == nil {
		return nil
	}
	out := new(EurekaConnectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EurekaConnectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EurekaSpec) DeepCopyInto(out *EurekaSpec) {
	*out = *in
	in.SyncToK8S.DeepCopyInto(&out.SyncToK8S)
	in.SyncFromK8S.DeepCopyInto(&out.SyncFromK8S)
	if in.Limiter != nil {
		in, out := &in.Limiter, &out.Limiter
		*out = new(Limiter)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EurekaSpec.
func (in *EurekaSpec) DeepCopy() *EurekaSpec {
	if in == nil {
		return nil
	}
	out := new(EurekaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EurekaStatus) DeepCopyInto(out *EurekaStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EurekaStatus.
func (in *EurekaStatus) DeepCopy() *EurekaStatus {
	if in == nil {
		return nil
	}
	out := new(EurekaStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EurekaSyncFromK8SSpec) DeepCopyInto(out *EurekaSyncFromK8SSpec) {
	*out = *in
	if in.AppendMetadatas != nil {
		in, out := &in.AppendMetadatas, &out.AppendMetadatas
		*out = make([]Metadata, len(*in))
		copy(*out, *in)
	}
	if in.AllowK8sNamespaces != nil {
		in, out := &in.AllowK8sNamespaces, &out.AllowK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DenyK8sNamespaces != nil {
		in, out := &in.DenyK8sNamespaces, &out.DenyK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EurekaSyncFromK8SSpec.
func (in *EurekaSyncFromK8SSpec) DeepCopy() *EurekaSyncFromK8SSpec {
	if in == nil {
		return nil
	}
	out := new(EurekaSyncFromK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EurekaSyncToK8SSpec) DeepCopyInto(out *EurekaSyncToK8SSpec) {
	*out = *in
	if in.FilterMetadatas != nil {
		in, out := &in.FilterMetadatas, &out.FilterMetadatas
		*out = make([]Metadata, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EurekaSyncToK8SSpec.
func (in *EurekaSyncToK8SSpec) DeepCopy() *EurekaSyncToK8SSpec {
	if in == nil {
		return nil
	}
	out := new(EurekaSyncToK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatewayConnector) DeepCopyInto(out *GatewayConnector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatewayConnector.
func (in *GatewayConnector) DeepCopy() *GatewayConnector {
	if in == nil {
		return nil
	}
	out := new(GatewayConnector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GatewayConnector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatewayConnectorList) DeepCopyInto(out *GatewayConnectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GatewayConnector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatewayConnectorList.
func (in *GatewayConnectorList) DeepCopy() *GatewayConnectorList {
	if in == nil {
		return nil
	}
	out := new(GatewayConnectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GatewayConnectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatewaySpec) DeepCopyInto(out *GatewaySpec) {
	*out = *in
	out.Ingress = in.Ingress
	out.Egress = in.Egress
	in.SyncToFgw.DeepCopyInto(&out.SyncToFgw)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatewaySpec.
func (in *GatewaySpec) DeepCopy() *GatewaySpec {
	if in == nil {
		return nil
	}
	out := new(GatewaySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GatewayStatus) DeepCopyInto(out *GatewayStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GatewayStatus.
func (in *GatewayStatus) DeepCopy() *GatewayStatus {
	if in == nil {
		return nil
	}
	out := new(GatewayStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressSelectorSpec) DeepCopyInto(out *IngressSelectorSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressSelectorSpec.
func (in *IngressSelectorSpec) DeepCopy() *IngressSelectorSpec {
	if in == nil {
		return nil
	}
	out := new(IngressSelectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Limiter) DeepCopyInto(out *Limiter) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Limiter.
func (in *Limiter) DeepCopy() *Limiter {
	if in == nil {
		return nil
	}
	out := new(Limiter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineConnector) DeepCopyInto(out *MachineConnector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineConnector.
func (in *MachineConnector) DeepCopy() *MachineConnector {
	if in == nil {
		return nil
	}
	out := new(MachineConnector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineConnector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineConnectorList) DeepCopyInto(out *MachineConnectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MachineConnector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineConnectorList.
func (in *MachineConnectorList) DeepCopy() *MachineConnectorList {
	if in == nil {
		return nil
	}
	out := new(MachineConnectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineConnectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSpec) DeepCopyInto(out *MachineSpec) {
	*out = *in
	out.SyncToK8S = in.SyncToK8S
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSpec.
func (in *MachineSpec) DeepCopy() *MachineSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineStatus) DeepCopyInto(out *MachineStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineStatus.
func (in *MachineStatus) DeepCopy() *MachineStatus {
	if in == nil {
		return nil
	}
	out := new(MachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSyncToK8SSpec) DeepCopyInto(out *MachineSyncToK8SSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSyncToK8SSpec.
func (in *MachineSyncToK8SSpec) DeepCopy() *MachineSyncToK8SSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSyncToK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metadata) DeepCopyInto(out *Metadata) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metadata.
func (in *Metadata) DeepCopy() *Metadata {
	if in == nil {
		return nil
	}
	out := new(Metadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosAuthSpec) DeepCopyInto(out *NacosAuthSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosAuthSpec.
func (in *NacosAuthSpec) DeepCopy() *NacosAuthSpec {
	if in == nil {
		return nil
	}
	out := new(NacosAuthSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosConnector) DeepCopyInto(out *NacosConnector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosConnector.
func (in *NacosConnector) DeepCopy() *NacosConnector {
	if in == nil {
		return nil
	}
	out := new(NacosConnector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NacosConnector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosConnectorList) DeepCopyInto(out *NacosConnectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NacosConnector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosConnectorList.
func (in *NacosConnectorList) DeepCopy() *NacosConnectorList {
	if in == nil {
		return nil
	}
	out := new(NacosConnectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NacosConnectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosSpec) DeepCopyInto(out *NacosSpec) {
	*out = *in
	out.Auth = in.Auth
	in.SyncToK8S.DeepCopyInto(&out.SyncToK8S)
	in.SyncFromK8S.DeepCopyInto(&out.SyncFromK8S)
	if in.Limiter != nil {
		in, out := &in.Limiter, &out.Limiter
		*out = new(Limiter)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosSpec.
func (in *NacosSpec) DeepCopy() *NacosSpec {
	if in == nil {
		return nil
	}
	out := new(NacosSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosStatus) DeepCopyInto(out *NacosStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosStatus.
func (in *NacosStatus) DeepCopy() *NacosStatus {
	if in == nil {
		return nil
	}
	out := new(NacosStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosSyncFromK8SSpec) DeepCopyInto(out *NacosSyncFromK8SSpec) {
	*out = *in
	if in.AppendMetadatas != nil {
		in, out := &in.AppendMetadatas, &out.AppendMetadatas
		*out = make([]Metadata, len(*in))
		copy(*out, *in)
	}
	if in.AllowK8sNamespaces != nil {
		in, out := &in.AllowK8sNamespaces, &out.AllowK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DenyK8sNamespaces != nil {
		in, out := &in.DenyK8sNamespaces, &out.DenyK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosSyncFromK8SSpec.
func (in *NacosSyncFromK8SSpec) DeepCopy() *NacosSyncFromK8SSpec {
	if in == nil {
		return nil
	}
	out := new(NacosSyncFromK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NacosSyncToK8SSpec) DeepCopyInto(out *NacosSyncToK8SSpec) {
	*out = *in
	if in.FilterMetadatas != nil {
		in, out := &in.FilterMetadatas, &out.FilterMetadatas
		*out = make([]Metadata, len(*in))
		copy(*out, *in)
	}
	if in.ClusterSet != nil {
		in, out := &in.ClusterSet, &out.ClusterSet
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.GroupSet != nil {
		in, out := &in.GroupSet, &out.GroupSet
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NacosSyncToK8SSpec.
func (in *NacosSyncToK8SSpec) DeepCopy() *NacosSyncToK8SSpec {
	if in == nil {
		return nil
	}
	out := new(NacosSyncToK8SSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncToFgwSpec) DeepCopyInto(out *SyncToFgwSpec) {
	*out = *in
	if in.AllowK8sNamespaces != nil {
		in, out := &in.AllowK8sNamespaces, &out.AllowK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DenyK8sNamespaces != nil {
		in, out := &in.DenyK8sNamespaces, &out.DenyK8sNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncToFgwSpec.
func (in *SyncToFgwSpec) DeepCopy() *SyncToFgwSpec {
	if in == nil {
		return nil
	}
	out := new(SyncToFgwSpec)
	in.DeepCopyInto(out)
	return out
}
