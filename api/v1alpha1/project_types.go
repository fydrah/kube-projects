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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LimitRangeSpec override corev1 LimitRangeSpec to add omitempty
// json tag (required in this case since limits are not required)
type LimitRangeSpec struct {
	Limits []v1.LimitRangeItem `json:"limits,omitempty" protobuf:"bytes,1,rep,name=limits"`
}

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// LimitRange is the Kubernetes limit range specs associated to the project
	LimitRange LimitRangeSpec `json:"limitRange,omitempty"`
	// ResourceQuota is the Kubernetes resource quota specs associated to the project
	ResourceQuota v1.ResourceQuotaSpec `json:"resourceQuota,omitempty"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Namespace is the namespace created by the project
	Namespace     v1.ObjectReference `json:"namespace,omitempty"`
	LimitRange    v1.ObjectReference `json:"limitRange,omitempty"`
	ResourceQuota v1.ObjectReference `json:"resourceQuota,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
