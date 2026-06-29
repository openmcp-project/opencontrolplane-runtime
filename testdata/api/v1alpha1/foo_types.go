/*
Copyright 2025.

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
	"k8s.io/apimachinery/pkg/runtime"
)

// FooSpec defines the desired state of Foo
type FooSpec struct {
	// +optional
	Foo *string `json:"foo,omitempty"`
}

// FooStatus defines the observed state of Foo.
type FooStatus struct {
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ObservedGeneration is the generation of this resource that was last reconciled by the controller.
	ObservedGeneration int64 `json:"observedGeneration"`
	// Phase is the current phase of the resource.
	Phase string `json:"phase"`
}

// Foo is the Schema for the foo API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=`.status.phase`,name="Phase",type=string
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:metadata:labels="openmcp.cloud/cluster=onboarding"
type Foo struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of FooService
	// +required
	Spec FooSpec `json:"spec"`

	// status defines the observed state of FooService
	// +optional
	Status FooStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// FooList contains a list of Foos
type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Foo `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &Foo{}, &FooList{})
		return nil
	})
}

// Finalizer returns the finalizer string for the FooService resource
func (o *Foo) Finalizer() string {
	return GroupVersion.Group + "/finalizer"
}

// GetStatus returns the status of the FooService resource
func (o *Foo) GetStatus() any {
	return o.Status
}

// GetConditions returns the conditions of the FooService resource
func (o *Foo) GetConditions() *[]metav1.Condition {
	return &o.Status.Conditions
}

// SetPhase sets the phase of the FooService resource status
func (o *Foo) SetPhase(phase string) {
	o.Status.Phase = phase
}

// SetObservedGeneration sets the observed generation of the FooService resource
func (o *Foo) SetObservedGeneration(gen int64) {
	o.Status.ObservedGeneration = gen
}
