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

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type NamedDeploymentSpec struct {
	Template appsv1.DeploymentSpec `json:"template"`
	Name string `json:"name"`
}


// InternalInNetworkTelemetryDeploymentSpec defines the desired state of InternalInNetworkTelemetryDeployment
type InternalInNetworkTelemetryDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DeploymentTemplates []NamedDeploymentSpec `json:"deployments"`
	CollectorRef v1.LocalObjectReference `json:"collectorRef"`
	CollectionId string `json:"collectionId"`
}

// InternalInNetworkTelemetryDeploymentStatus defines the observed state of InternalInNetworkTelemetryDeployment
type InternalInNetworkTelemetryDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InternalInNetworkTelemetryDeployment is the Schema for the internalinnetworktelemetrydeployments API
type InternalInNetworkTelemetryDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InternalInNetworkTelemetryDeploymentSpec   `json:"spec,omitempty"`
	Status InternalInNetworkTelemetryDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InternalInNetworkTelemetryDeploymentList contains a list of InternalInNetworkTelemetryDeployment
type InternalInNetworkTelemetryDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InternalInNetworkTelemetryDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InternalInNetworkTelemetryDeployment{}, &InternalInNetworkTelemetryDeploymentList{})
}
