/*
Copyright 2025 NTT DATA.

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

// Define constants for RDS instance state values
const (
	StateCreating = "creating"
	StateUpdating = "updating"
	StateDeleting = "deleting"
	StateCreated  = "created"
	StateUpdated  = "updated"
	StateDeleted  = "deleted"
)

// AwsRDSDemoInstanceSpec defines the desired state of AwsRDSDemoInstance
type AwsRDSDemoInstanceSpec struct {
	DBInstanceClass string `json:"dbInstanceClass"`
	Engine          string `json:"engine"`
	EngineVersion   string `json:"engineVersion"`
	DBName          string `json:"dbName"`
	// +kubebuilder:validation:Minimum=20
	AllocatedStorage     int32  `json:"allocatedStorage"`
	Stage                string `json:"stage"`
	CredentialSecretName string `json:"credentialSecretName"`
}

// AwsRDSDemoInstanceStatus defines the observed state of AwsRDSDemoInstance
type AwsRDSDemoInstanceStatus struct {
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AwsRDSDemoInstance is the Schema for the awsrdsdemoinstances API
type AwsRDSDemoInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AwsRDSDemoInstanceSpec   `json:"spec,omitempty"`
	Status AwsRDSDemoInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AwsRDSDemoInstanceList contains a list of AwsRDSDemoInstance
type AwsRDSDemoInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AwsRDSDemoInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AwsRDSDemoInstance{}, &AwsRDSDemoInstanceList{})
}
