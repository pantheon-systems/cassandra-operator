package nodetool

// NodeState represents the phase/state of the cassandra node in the ring
type NodeState string

// NodeStates enumerated
const (
	NodeStateUnknown NodeState = "Unknown"
	NodeStateNormal  NodeState = "Normal"
	NodeStateLeaving NodeState = "Leaving"
	NodeStateJoining NodeState = "Joining"
	NodeStateMoving  NodeState = "Moving"
)

// NodeStatus represents the status of the cassandra node in the ring
type NodeStatus string

// NodeStatus enumerated
const (
	NodeStatusUnknown NodeStatus = "Unknown"
	NodeStatusUp      NodeStatus = "Up"
	NodeStatusDown    NodeStatus = "Down"
)

// NodeMode represents the mode of the cassandra node as reported by netstats
type NodeMode string

// NodeMode enumerated
const (
	NodeModeJoining        NodeMode = "JOINING"
	NodeModeLeaving        NodeMode = "LEAVING"
	NodeModeNormal         NodeMode = "NORMAL"
	NodeModeDecommissioned NodeMode = "DECOMMISSIONED"
	NodeModeClient         NodeMode = "CLIENT"
)
