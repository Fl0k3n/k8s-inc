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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type LinkType string

const (
	NODE LinkType = "node"
	GENERIC_NETWORK_DEVICE LinkType = "net"
	INC_NETWORK_DEVICE LinkType = "inc"
)

type Link struct {
	To LinkType `json:"type"`
	PeerName string `json:"peerName"`
	Port int32 `json:"port"`
	Ipv4 string `json:"ipv4"`
}

// IncSwitchSpec defines the desired state of IncSwitch
type IncSwitchSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of IncSwitch. Edit incswitch_types.go to remove/update
	Arch string `json:"arch"`
	ControllerPort int32 `json:"controllerPort"`
	P4RuntimePort int32 `json:"p4RuntimePort"`
	Links []Link `json:"links,omitempty"`
	MyOptionalField string `json:"myOptionalField,omitempty"`
}

// IncSwitchStatus defines the observed state of IncSwitch
type IncSwitchStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	MyOwnStatusField string `json:"myOwnStatusField,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IncSwitch is the Schema for the incswitches API
type IncSwitch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IncSwitchSpec   `json:"spec,omitempty"`
	Status IncSwitchStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IncSwitchList contains a list of IncSwitch
type IncSwitchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IncSwitch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IncSwitch{}, &IncSwitchList{})
}
