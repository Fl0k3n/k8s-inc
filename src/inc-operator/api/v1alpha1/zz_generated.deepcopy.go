//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2024.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeployment) DeepCopyInto(out *ExternalInNetworkTelemetryDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeployment.
func (in *ExternalInNetworkTelemetryDeployment) DeepCopy() *ExternalInNetworkTelemetryDeployment {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeploymentList) DeepCopyInto(out *ExternalInNetworkTelemetryDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExternalInNetworkTelemetryDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeploymentList.
func (in *ExternalInNetworkTelemetryDeploymentList) DeepCopy() *ExternalInNetworkTelemetryDeploymentList {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeploymentSpec) DeepCopyInto(out *ExternalInNetworkTelemetryDeploymentSpec) {
	*out = *in
	in.DeploymentTemplate.DeepCopyInto(&out.DeploymentTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeploymentSpec.
func (in *ExternalInNetworkTelemetryDeploymentSpec) DeepCopy() *ExternalInNetworkTelemetryDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryDeploymentStatus) DeepCopyInto(out *ExternalInNetworkTelemetryDeploymentStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryDeploymentStatus.
func (in *ExternalInNetworkTelemetryDeploymentStatus) DeepCopy() *ExternalInNetworkTelemetryDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpoints) DeepCopyInto(out *ExternalInNetworkTelemetryEndpoints) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpoints.
func (in *ExternalInNetworkTelemetryEndpoints) DeepCopy() *ExternalInNetworkTelemetryEndpoints {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpoints)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryEndpoints) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsEntry) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsEntry) {
	*out = *in
	out.PodReference = in.PodReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsEntry.
func (in *ExternalInNetworkTelemetryEndpointsEntry) DeepCopy() *ExternalInNetworkTelemetryEndpointsEntry {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsEntry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsList) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExternalInNetworkTelemetryEndpoints, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsList.
func (in *ExternalInNetworkTelemetryEndpointsList) DeepCopy() *ExternalInNetworkTelemetryEndpointsList {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExternalInNetworkTelemetryEndpointsList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsSpec) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsSpec) {
	*out = *in
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]ExternalInNetworkTelemetryEndpointsEntry, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsSpec.
func (in *ExternalInNetworkTelemetryEndpointsSpec) DeepCopy() *ExternalInNetworkTelemetryEndpointsSpec {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalInNetworkTelemetryEndpointsStatus) DeepCopyInto(out *ExternalInNetworkTelemetryEndpointsStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalInNetworkTelemetryEndpointsStatus.
func (in *ExternalInNetworkTelemetryEndpointsStatus) DeepCopy() *ExternalInNetworkTelemetryEndpointsStatus {
	if in == nil {
		return nil
	}
	out := new(ExternalInNetworkTelemetryEndpointsStatus)
	in.DeepCopyInto(out)
	return out
}
