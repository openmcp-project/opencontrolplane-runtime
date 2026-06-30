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

// FooServiceSpec defines the desired state of FooService
type FooServiceSpec struct {
	// foo is an example field of FooService. Edit fooservice_types.go to remove/update
	// +optional
	Foo *string `json:"foo,omitempty"`
}

// FooServiceStatus defines the observed state of FooService.
type FooServiceStatus struct {
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

// FooService is the Schema for the fooservices API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=`.status.phase`,name="Phase",type=string
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:metadata:labels="openmcp.cloud/cluster=onboarding"
type FooService struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of FooService
	// +required
	Spec FooServiceSpec `json:"spec"`

	// status defines the observed state of FooService
	// +optional
	Status FooServiceStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// FooServiceList contains a list of FooService
type FooServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FooService `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &FooService{}, &FooServiceList{})
		return nil
	})
}

// Finalizer returns the finalizer string for the FooService resource
func (o *FooService) Finalizer() string {
	return GroupVersion.Group + "/finalizer"
}

// GetStatus returns the status of the FooService resource
func (o *FooService) GetStatus() any {
	return o.Status
}

// GetConditions returns the conditions of the FooService resource
func (o *FooService) GetConditions() *[]metav1.Condition {
	return &o.Status.Conditions
}

// SetPhase sets the phase of the FooService resource status
func (o *FooService) SetPhase(phase string) {
	o.Status.Phase = phase
}

// SetObservedGeneration sets the observed generation of the FooService resource
func (o *FooService) SetObservedGeneration(gen int64) {
	o.Status.ObservedGeneration = gen
}
