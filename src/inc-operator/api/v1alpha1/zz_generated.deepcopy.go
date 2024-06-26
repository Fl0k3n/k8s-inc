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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Collector) DeepCopyInto(out *Collector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Collector.
func (in *Collector) DeepCopy() *Collector {
	if in == nil {
		return nil
	}
	out := new(Collector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Collector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CollectorList) DeepCopyInto(out *CollectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Collector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CollectorList.
func (in *CollectorList) DeepCopy() *CollectorList {
	if in == nil {
		return nil
	}
	out := new(CollectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CollectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CollectorSpec) DeepCopyInto(out *CollectorSpec) {
	*out = *in
	in.PodSpec.DeepCopyInto(&out.PodSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CollectorSpec.
func (in *CollectorSpec) DeepCopy() *CollectorSpec {
	if in == nil {
		return nil
	}
	out := new(CollectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CollectorStatus) DeepCopyInto(out *CollectorStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeRef != nil {
		in, out := &in.NodeRef, &out.NodeRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
	if in.ReportingPort != nil {
		in, out := &in.ReportingPort, &out.ReportingPort
		*out = new(int32)
		**out = **in
	}
	if in.ApiServiceRef != nil {
		in, out := &in.ApiServiceRef, &out.ApiServiceRef
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CollectorStatus.
func (in *CollectorStatus) DeepCopy() *CollectorStatus {
	if in == nil {
		return nil
	}
	out := new(CollectorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentEndpoints) DeepCopyInto(out *DeploymentEndpoints) {
	*out = *in
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]InternalInNetworkTelemetryEndpointsEntry, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentEndpoints.
func (in *DeploymentEndpoints) DeepCopy() *DeploymentEndpoints {
	if in == nil {
		return nil
	}
	out := new(DeploymentEndpoints)
	in.DeepCopyInto(out)
	return out
}

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
	in.IngressInfo.DeepCopyInto(&out.IngressInfo)
	if in.RequireAtLeastIntDevices != nil {
		in, out := &in.RequireAtLeastIntDevices, &out.RequireAtLeastIntDevices
		*out = new(string)
		**out = **in
	}
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
	in.Status.DeepCopyInto(&out.Status)
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
	in.IngressInfo.DeepCopyInto(&out.IngressInfo)
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
	if in.ExternalIngressInfo != nil {
		in, out := &in.ExternalIngressInfo, &out.ExternalIngressInfo
		*out = new(IngressInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.ExternalMonitoringPolicy != nil {
		in, out := &in.ExternalMonitoringPolicy, &out.ExternalMonitoringPolicy
		*out = new(MonitoringPolicy)
		**out = **in
	}
	if in.InternalIngressInfo != nil {
		in, out := &in.InternalIngressInfo, &out.InternalIngressInfo
		*out = new(IngressInfo)
		(*in).DeepCopyInto(*out)
	}
	if in.InternalMonitoringPolicy != nil {
		in, out := &in.InternalMonitoringPolicy, &out.InternalMonitoringPolicy
		*out = new(MonitoringPolicy)
		**out = **in
	}
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

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressInfo) DeepCopyInto(out *IngressInfo) {
	*out = *in
	if in.NodeNames != nil {
		in, out := &in.NodeNames, &out.NodeNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressInfo.
func (in *IngressInfo) DeepCopy() *IngressInfo {
	if in == nil {
		return nil
	}
	out := new(IngressInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeployment) DeepCopyInto(out *InternalInNetworkTelemetryDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeployment.
func (in *InternalInNetworkTelemetryDeployment) DeepCopy() *InternalInNetworkTelemetryDeployment {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeploymentList) DeepCopyInto(out *InternalInNetworkTelemetryDeploymentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InternalInNetworkTelemetryDeployment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeploymentList.
func (in *InternalInNetworkTelemetryDeploymentList) DeepCopy() *InternalInNetworkTelemetryDeploymentList {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeploymentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryDeploymentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeploymentSpec) DeepCopyInto(out *InternalInNetworkTelemetryDeploymentSpec) {
	*out = *in
	if in.DeploymentTemplates != nil {
		in, out := &in.DeploymentTemplates, &out.DeploymentTemplates
		*out = make([]NamedDeploymentSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.CollectorRef = in.CollectorRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeploymentSpec.
func (in *InternalInNetworkTelemetryDeploymentSpec) DeepCopy() *InternalInNetworkTelemetryDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryDeploymentStatus) DeepCopyInto(out *InternalInNetworkTelemetryDeploymentStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryDeploymentStatus.
func (in *InternalInNetworkTelemetryDeploymentStatus) DeepCopy() *InternalInNetworkTelemetryDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpoints) DeepCopyInto(out *InternalInNetworkTelemetryEndpoints) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpoints.
func (in *InternalInNetworkTelemetryEndpoints) DeepCopy() *InternalInNetworkTelemetryEndpoints {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpoints)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryEndpoints) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsEntry) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsEntry) {
	*out = *in
	out.PodReference = in.PodReference
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsEntry.
func (in *InternalInNetworkTelemetryEndpointsEntry) DeepCopy() *InternalInNetworkTelemetryEndpointsEntry {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsEntry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsList) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]InternalInNetworkTelemetryEndpoints, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsList.
func (in *InternalInNetworkTelemetryEndpointsList) DeepCopy() *InternalInNetworkTelemetryEndpointsList {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InternalInNetworkTelemetryEndpointsList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsSpec) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsSpec) {
	*out = *in
	if in.DeploymentEndpoints != nil {
		in, out := &in.DeploymentEndpoints, &out.DeploymentEndpoints
		*out = make([]DeploymentEndpoints, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.CollectorRef = in.CollectorRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsSpec.
func (in *InternalInNetworkTelemetryEndpointsSpec) DeepCopy() *InternalInNetworkTelemetryEndpointsSpec {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalInNetworkTelemetryEndpointsStatus) DeepCopyInto(out *InternalInNetworkTelemetryEndpointsStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalInNetworkTelemetryEndpointsStatus.
func (in *InternalInNetworkTelemetryEndpointsStatus) DeepCopy() *InternalInNetworkTelemetryEndpointsStatus {
	if in == nil {
		return nil
	}
	out := new(InternalInNetworkTelemetryEndpointsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metrics) DeepCopyInto(out *Metrics) {
	*out = *in
	if in.DeviceMetrics != nil {
		in, out := &in.DeviceMetrics, &out.DeviceMetrics
		*out = make([]SwitchMetrics, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metrics.
func (in *Metrics) DeepCopy() *Metrics {
	if in == nil {
		return nil
	}
	out := new(Metrics)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricsSummary) DeepCopyInto(out *MetricsSummary) {
	*out = *in
	in.WindowMetrics.DeepCopyInto(&out.WindowMetrics)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricsSummary.
func (in *MetricsSummary) DeepCopy() *MetricsSummary {
	if in == nil {
		return nil
	}
	out := new(MetricsSummary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamedDeploymentSpec) DeepCopyInto(out *NamedDeploymentSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamedDeploymentSpec.
func (in *NamedDeploymentSpec) DeepCopy() *NamedDeploymentSpec {
	if in == nil {
		return nil
	}
	out := new(NamedDeploymentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PortMetrics) DeepCopyInto(out *PortMetrics) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PortMetrics.
func (in *PortMetrics) DeepCopy() *PortMetrics {
	if in == nil {
		return nil
	}
	out := new(PortMetrics)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SwitchMetrics) DeepCopyInto(out *SwitchMetrics) {
	*out = *in
	if in.PortMetrics != nil {
		in, out := &in.PortMetrics, &out.PortMetrics
		*out = make([]PortMetrics, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SwitchMetrics.
func (in *SwitchMetrics) DeepCopy() *SwitchMetrics {
	if in == nil {
		return nil
	}
	out := new(SwitchMetrics)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryCollectionStats) DeepCopyInto(out *TelemetryCollectionStats) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryCollectionStats.
func (in *TelemetryCollectionStats) DeepCopy() *TelemetryCollectionStats {
	if in == nil {
		return nil
	}
	out := new(TelemetryCollectionStats)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TelemetryCollectionStats) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryCollectionStatsList) DeepCopyInto(out *TelemetryCollectionStatsList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TelemetryCollectionStats, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryCollectionStatsList.
func (in *TelemetryCollectionStatsList) DeepCopy() *TelemetryCollectionStatsList {
	if in == nil {
		return nil
	}
	out := new(TelemetryCollectionStatsList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TelemetryCollectionStatsList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryCollectionStatsSpec) DeepCopyInto(out *TelemetryCollectionStatsSpec) {
	*out = *in
	out.CollectorRef = in.CollectorRef
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryCollectionStatsSpec.
func (in *TelemetryCollectionStatsSpec) DeepCopy() *TelemetryCollectionStatsSpec {
	if in == nil {
		return nil
	}
	out := new(TelemetryCollectionStatsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryCollectionStatsStatus) DeepCopyInto(out *TelemetryCollectionStatsStatus) {
	*out = *in
	if in.MetricsSummary != nil {
		in, out := &in.MetricsSummary, &out.MetricsSummary
		*out = new(MetricsSummary)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryCollectionStatsStatus.
func (in *TelemetryCollectionStatsStatus) DeepCopy() *TelemetryCollectionStatsStatus {
	if in == nil {
		return nil
	}
	out := new(TelemetryCollectionStatsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryDeviceResource) DeepCopyInto(out *TelemetryDeviceResource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryDeviceResource.
func (in *TelemetryDeviceResource) DeepCopy() *TelemetryDeviceResource {
	if in == nil {
		return nil
	}
	out := new(TelemetryDeviceResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryDeviceResources) DeepCopyInto(out *TelemetryDeviceResources) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryDeviceResources.
func (in *TelemetryDeviceResources) DeepCopy() *TelemetryDeviceResources {
	if in == nil {
		return nil
	}
	out := new(TelemetryDeviceResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TelemetryDeviceResources) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryDeviceResourcesList) DeepCopyInto(out *TelemetryDeviceResourcesList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TelemetryDeviceResources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryDeviceResourcesList.
func (in *TelemetryDeviceResourcesList) DeepCopy() *TelemetryDeviceResourcesList {
	if in == nil {
		return nil
	}
	out := new(TelemetryDeviceResourcesList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TelemetryDeviceResourcesList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryDeviceResourcesSpec) DeepCopyInto(out *TelemetryDeviceResourcesSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryDeviceResourcesSpec.
func (in *TelemetryDeviceResourcesSpec) DeepCopy() *TelemetryDeviceResourcesSpec {
	if in == nil {
		return nil
	}
	out := new(TelemetryDeviceResourcesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TelemetryDeviceResourcesStatus) DeepCopyInto(out *TelemetryDeviceResourcesStatus) {
	*out = *in
	if in.DeviceResources != nil {
		in, out := &in.DeviceResources, &out.DeviceResources
		*out = make([]TelemetryDeviceResource, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TelemetryDeviceResourcesStatus.
func (in *TelemetryDeviceResourcesStatus) DeepCopy() *TelemetryDeviceResourcesStatus {
	if in == nil {
		return nil
	}
	out := new(TelemetryDeviceResourcesStatus)
	in.DeepCopyInto(out)
	return out
}
