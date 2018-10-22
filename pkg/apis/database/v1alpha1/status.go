package v1alpha1

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

// action (create cluster)
//    P:Initial S:Initial ->
//    P: Creating S: Bootstrap(Create services and service account and stuff) ->
//    P: Creating S: Scale(Node1) ->
//    P: Creating S: Join(Node1) ->
//    P: Creating S: Scale(2) ->
//    P: Creating S: Join(Node2) ->
//    P: Running  S: Run()
//
// action (create cluster failure)
//    P: Initial S:Initial ->
//    P: Creating S: Scale(Node1) ->
//    P: Failed   S: ScaleFailed -> retry
//
// action (scale by one)
//    P: Running  S: Scale(3)
//    P: Running  S: Join(Node3)
//    P: Running  S: Repair()
//    P: Running  S: Run()
//
// non-action (single node is failed)
//    P: Degraded  S: ProbeFailed(nodeDown)  Assumes probe retry et-al detects failure on a node
//
// action (scale down 1)
//    P: Degraded  S: Drain(Node3) ->
//    P: Degraded  S: Decomissioning(Node3)
//    P: Degraded  S: Repair()
//    P: Running   S: Run()
//
// action (delete cluster)
//    P: Terminating S: Drain(Node2) ->
//    P: Terminating S: Decomissioning(Node2)
//    P: Terminating S: Drain(Node1) ->
//    P: Terminating S: Decomissioning(Node1)
//    P: Deleted     S: Deleted

// C(Cluster)  -> SS  -> Pods
//             -> CN(CassaNode Controller Single node)
//             ->  CN P: S:
//
// ClusterPhase represents the phase/state of the cluster

// ClusterStatus specifies the status of the cassandra cluster
type ClusterStatus struct {
	Phase          ClusterPhase `json:"phase"`
	State          ClusterState `json:"state"`
	Members        NodesStatus  `json:"members"`
	CurrentVersion string       `json:"currentVersion"`
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
