package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultCassandraImage Default repository for cassandra images
	DefaultCassandraImage = "quay.io/getpantheon/cassandra"
	// DefaultCassandraTag Default tag/version for cassandra image
	DefaultCassandraTag = "2x-64"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CassandraClusterList Lists of CassandraClusters
type CassandraClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `jsoån:"metadata,omitempty"`
	Items           []CassandraCluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CassandraCluster CassandraCluster api representation
type CassandraCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

// ClusterSpec Specification for cassandra cluster for API
type ClusterSpec struct {
	Size                      int              `json:"size"`
	Repair                    *RepairPolicy    `json:"repair,omitempty"`
	Node                      *NodePolicy      `json:"node"`
	KeyspaceName              string           `json:"keyspaceName,omitempty"`
	SecretName                string           `json:"secretName,omitempty"`
	ConfigMapName             string           `json:"configMapName,omitempty"`
	JvmAgentConfigName        string           `json:"jvmAgentConfigName,omitemtpy"`
	JvmAgent                  string           `json:"jvmAgent,omitempty"`
	Datacenter                string           `json:"datacenter"`
	ExternalSeeds             []string         `json:"externalSeeds,omitempty"`
	EnablePublicPodServices   bool             `json:"enablePublicPodServices"`
	ExposePublicLB            bool             `json:"exposePublicLB"`
	EnablePodDisruptionBudget bool             `json:"enablePodDisruptionBudget"`
	Affinity                  *corev1.Affinity `json:"affinity,omitempty"`
}

// RepairPolicy sets the policies for the automated cassandra repair job
type RepairPolicy struct {
	Schedule string `json:"schedule"`
	Image    string `json:"image,omitempty"`
}

// NodePolicy specifies the details of constructing a cassandra node
type NodePolicy struct {
	Resources        *corev1.ResourceRequirements `json:"resources,omitempty"`
	PersistentVolume *PersistentVolumeSpec        `json:"persistentVolume,omitempty"`
	Image            string                       `json:"image"`
	FileMountPath    string                       `json:"fileMountPath"`
}

// PersistentVolumeSpec exposes configurables for the PV for the stateful set
type PersistentVolumeSpec struct {
	StorageClassName string              `json:"storageClass,omitempty"`
	Capacity         corev1.ResourceList `json:"resources,omitempty"`
}
