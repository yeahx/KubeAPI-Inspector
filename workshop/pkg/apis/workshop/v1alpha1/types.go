package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ClusterSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Cluster `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	Tenant     string `json:"tenant,omitempty" protobuf:"bytes,1,opt,name=tenant"`
	Name       string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	APIServer  string `json:"apiserver,omitempty" protobuf:"bytes,3,opt,name=apiserver"`
	Kubeconfig []byte `json:"kubeconfig,omitempty" protobuf:"bytes,4,opt,name=kubeconfig"`
}

// +kubebuilder:subresource:status

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
}
