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

// ClusterPhase type alias for the string representing the phase
type ClusterPhase string

// ClusterPhases enumerated
const (
	// ClusterPhaseInitial is the initial phase before resources are created
	// but after the CRD resource is created in kube
	ClusterPhaseInitial ClusterPhase = "Initial"
	// ClusterPhaseCreating is the second phase after resources (pods,services) are created
	// but before there are active pods
	ClusterPhaseCreating ClusterPhase = "Creating"
	// ClusterPhaseInitializing is after the pods are active but before they have fully joined
	// this fase is pod startup, cassandra startup, cassandra joining
	ClusterPhaseInitializing ClusterPhase = "Initializing"
	// ClusterPhaseRunning pods are all joined and normal
	ClusterPhaseRunning ClusterPhase = "Running"
	// ClusterPhaseScaling occurs when a pod/cassandra node is leaving or joining the ring, inclusive of pod creation
	// and pod startup
	ClusterPhaseScaling ClusterPhase = "Scaling"
	// ClusterPhaseFailed when the initial pod that is the seed creating the cluster fails to create
	ClusterPhaseFailed ClusterPhase = "Failed"
	// ClusterPhaseTerminated when the cluster is being deleted, when the child resources are is being deleted through
	// all child resources being gone
	ClusterPhaseTerminating ClusterPhase = "Terminating"
	// ClusterPhaseUnknown if the cluster is out of band with the statemachine, this is the state
	ClusterPhaseUnknown ClusterPhase = "Unknown"
)

// ClusterState represents a state in the cassandra cluster state-machine
type ClusterState string

var clusterStateDescription = map[ClusterState]string{}

// Describe returns a description of the cluster state
func (cs ClusterState) Describe() string {
	description := clusterStateDescription[cs]
	return description
}

const (
	// ClusterStateInitial Initial state of the cluster, no stateful set created
	ClusterStateInitial ClusterState = "Initial"
	// ClusterStateBootstrap Clusters first node is creating and starting up
	ClusterStateBootstrap ClusterState = "Bootstrap"
	// ClusterStateScale Cluster is scaling up/down by 1
	ClusterStateScale ClusterState = "Scale"
	// ClusterStateJoin Cluster is joining the ring
	ClusterStateJoin ClusterState = "Join"
	// ClusterStateRun Cluster is up and running
	ClusterStateRun ClusterState = "Run"
	// ClusterStateScaleFail Cluster is in a failed state
	ClusterStateScaleFail ClusterState = "ScaleFail"
	// ClusterStateRepair Cluster is executing a repair
	ClusterStateRepair ClusterState = "Repair"
	// ClusterStateDecomission Cluster is in process of decomissioning
	ClusterStateDecomission ClusterState = "Decomission"
	// ClusterStateProbeFail error state for cluster probe failing
	ClusterStateProbeFail ClusterState = "ProbeFail"
	// ClusterStateDelete Cluster has been deleted
	ClusterStateDelete ClusterState = "Delete"
)

func init() {
	SchemeBuilder.Register(&CassandraCluster{}, &CassandraClusterList{})
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CassandraCluster is the Schema for the cassandraclusters API
// +k8s:openapi-gen=true
type CassandraCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CassandraClusterList contains a list of CassandraCluster
type CassandraClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CassandraCluster `json:"items"`
}

// ClusterStatus defines the observed state of CassandraCluster
type ClusterStatus struct {
	Phase          ClusterPhase `json:"phase"`
	State          ClusterState `json:"state"`
	Members        NodesStatus  `json:"members"`
	CurrentVersion string       `json:"currentVersion"`
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

// NodesStatus bins nodes by state
type NodesStatus struct {
	Creating []string `json:"creating,omitempty"`
	Ready    []string `json:"ready,omitempty"`
	Joining  []string `json:"joining,omitempty"`
	Leaving  []string `json:"leaving,omitempty"`
	Unready  []string `json:"unready,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
}

// Provisioning returns true if the cluster has a node that is in process of provisioning
// (Joining, Creating)
func (s *ClusterStatus) Provisioning() bool {
	nodesInTrans := len(s.Members.Creating) + len(s.Members.Joining)
	return s.Phase == ClusterPhaseCreating || (s.Phase == ClusterPhaseInitializing && nodesInTrans > 0)
}

// NodesInTransit returns true if ANY cluster nodes are joining, creating or leaving the cluster
func (s *ClusterStatus) NodesInTransit() bool {
	return len(s.Members.Creating)+len(s.Members.Joining)+len(s.Members.Leaving) > 0
}
