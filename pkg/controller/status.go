package controller

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// nodeStatusReporter is an interface that constricts the nodeStatusReporter implentation
// needed behavior. So that we can better decouple this classes required contract vs the implementation
type nodeStatusReporter interface {
	GetStatus(node *corev1.Pod) (map[string]*nodetool.Status, error)
	GetHostID(node *corev1.Pod) (string, error)
}

// nodeStatusReporter is an interface that constricts the nodeStatusReporter implentation
// needed behavior. So that we can better decouple this classes required contract vs the implementation
type resourceListerUpdater interface {
	List(namespace string, into sdk.Object, opts ...sdk.ListOption) error
	Update(object sdk.Object) error
}

// ClusterStatusManager updates and calculates a clusters status
type ClusterStatusManager struct {
	nodeStatusReporter nodeStatusReporter
	listerUpdater      resourceListerUpdater
}

// NewStatusManager returns a new ClusterStatusController to get the status of the cluster
func NewStatusManager(statusReporter nodeStatusReporter, listerUpdater resourceListerUpdater) *ClusterStatusManager {
	return &ClusterStatusManager{
		nodeStatusReporter: statusReporter,
		listerUpdater:      listerUpdater,
	}
}

// Update calculates the current status and updates the k8s resource status accordingly
func (c *ClusterStatusManager) Update(cc *v1alpha1.CassandraCluster) error {
	currentStatus, err := c.getClusterStatus(cc)
	if err != nil {
		return err
	}

	currentStatus.DeepCopyInto(&cc.Status)
	return c.listerUpdater.Update(cc)
}

func (c *ClusterStatusManager) getClusterStatus(cc *v1alpha1.CassandraCluster) (*v1alpha1.ClusterStatus, error) {
	pods, err := c.getClusterPods(cc.GetName(), cc.GetNamespace(), cc.GetLabels())
	if err != nil {
		return nil, err
	}

	// we are unknown till we are known
	status := &v1alpha1.ClusterStatus{
		Phase: v1alpha1.ClusterPhaseUnknown,
	}

	currentStatus := cc.Status
	actualPodCount := len(pods.Items)

	// RULE: None/Initial and No Pods -> Initial
	if currentStatus.Phase == "" || (currentStatus.Phase == v1alpha1.ClusterPhaseInitial && actualPodCount == 0) {
		currentStatus.Phase = v1alpha1.ClusterPhaseInitial
		return &currentStatus, nil
	}

	// loop through pods and add to status buckets in status object
	kubeNodeStatuses, err := c.groupPodsByState(pods.Items)
	if err != nil {
		return nil, err
	}
	status.Members = *kubeNodeStatuses

	// we have not kicked off the creation
	// for the cluster
	if (currentStatus.Phase == v1alpha1.ClusterPhaseInitial || currentStatus.Phase == v1alpha1.ClusterPhaseCreating) &&
		actualPodCount > 0 && len(status.Members.Creating) > 0 {
		status.Phase = v1alpha1.ClusterPhaseCreating
		return status, nil
	}

	if len(status.Members.Unready) > 0 {
		// creating first node fails
		if currentStatus.Phase == v1alpha1.ClusterPhaseCreating ||
			currentStatus.Phase == v1alpha1.ClusterPhaseInitializing {
			status.Phase = v1alpha1.ClusterPhaseFailed
			return status, nil
		}

		// cluster is running or scaling and node fails
		// in this case the stateful set should handle the restart, self healing
		// we should keep track of this, if it is consistant or we get to crashloopbackoff
		// we should do something else though i am not sure what besides page the OCE
	}

	// we are creating, we have not yet created all the nodes that we are supposed to create
	// and we have more than one ready or one joining
	if currentStatus.Phase == v1alpha1.ClusterPhaseCreating {
		if actualPodCount != cc.Spec.Size &&
			(len(status.Members.Joining) == 1 || len(status.Members.Ready) > 0) {
			status.Phase = v1alpha1.ClusterPhaseInitializing
			return status, nil
		}
	}

	// we have one or more pods ready and a single pod either joining, creating or leaving, stay or move to Scaling
	if currentStatus.Phase == v1alpha1.ClusterPhaseRunning || currentStatus.Phase == v1alpha1.ClusterPhaseScaling {
		if len(status.Members.Ready) > 0 && (len(status.Members.Joining) == 1 || len(status.Members.Creating) == 1 || len(status.Members.Leaving) == 1) {
			status.Phase = v1alpha1.ClusterPhaseScaling
			return status, nil
		}
	}

	// All pods expected are ready and running/joined regardless of Phase
	if len(status.Members.Leaving) == 0 && cc.Spec.Size == len(status.Members.Ready) {
		status.Phase = v1alpha1.ClusterPhaseRunning
		return status, nil
	}

	// we are not yet ready and are in initializing state, so stay there
	if currentStatus.Phase == v1alpha1.ClusterPhaseInitializing {
		status.Phase = v1alpha1.ClusterPhaseInitializing
		return status, nil
	}

	// if we got here we are in an unkown state, this should eventually
	// be logged and processed by an engineer to add the state accordingly
	// Update current status in kube
	return status, nil
}

func (c *ClusterStatusManager) groupPodsByState(pods []corev1.Pod) (*v1alpha1.NodesStatus, error) {
	var nodeStatuses map[string]*nodetool.Status
	var err error

	nodeStates := &v1alpha1.NodesStatus{}
	for _, pod := range pods {
		podName := pod.GetName()

		if pod.DeletionTimestamp != nil {
			nodeStates.Deleted = append(nodeStates.Deleted, podName)
			continue
		}

		if pod.Status.Phase != corev1.PodPending {
			if len(pod.Status.ContainerStatuses) > 1 &&
				(pod.Status.ContainerStatuses[0].State.Terminated != nil ||
					pod.Status.ContainerStatuses[0].RestartCount > 0) {
				nodeStates.Unready = append(nodeStates.Unready, podName)
				continue
			}
		}

		switch pod.Status.Phase {
		case corev1.PodPending:
			nodeStates.Creating = append(nodeStates.Creating, podName)
			continue
		case corev1.PodFailed:
			nodeStates.Unready = append(nodeStates.Unready, podName)
			continue
		case corev1.PodUnknown:
			nodeStates.Unready = append(nodeStates.Unready, podName)
			continue
		}

		if pod.Status.Phase != corev1.PodRunning {
			// if the status is not one handeld by the switch
			// then we error if it is not running, this is to
			// prevent them from adding phases and us rolling forward
			// thinking its running
			return nil, fmt.Errorf("Unsupported PodPhase: %s", pod.Status.Phase)
		}

		// pod is corev1.Running, now check if it is ready
		readyCondition := corev1.ConditionUnknown
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				readyCondition = condition.Status
			}
		}

		if readyCondition != corev1.ConditionTrue {
			// running state but not ready means that it is waiting for the service
			// or is starting up, ie the healthcheck which uses nodetool does not
			// succeed
			nodeStates.Creating = append(nodeStates.Creating, podName)
			continue
		}

		// if it is not set, we fetch the nodetool status for all nodes
		// then memoize the result for the rest of the pods
		if len(nodeStatuses) == 0 {
			nodeStatuses, err = c.nodeStatusReporter.GetStatus(&pod)
			if err != nil {
				return nil, err
			}
		}

		// getting cassandra node id
		hostID, err := c.nodeStatusReporter.GetHostID(&pod)
		if err != nil {
			return nil, err
		}

		switch nodeStatuses[hostID].State {
		case nodetool.NodeStateJoining:
			nodeStates.Joining = append(nodeStates.Joining, podName)
		case nodetool.NodeStateNormal:
			nodeStates.Ready = append(nodeStates.Ready, podName)
		case nodetool.NodeStateLeaving:
			nodeStates.Leaving = append(nodeStates.Leaving, podName)
		default:
			nodeStates.Unready = append(nodeStates.Unready, podName)
		}
		continue
	}

	return nodeStates, nil
}

// GetClusterPods retrieves the pods for a specific cluster in a specific namespace
func (c *ClusterStatusManager) getClusterPods(clusterName, namespace string, clusterLabels map[string]string) (*corev1.PodList, error) {
	pods := &corev1.PodList{
		TypeMeta: resource.GetPodTypeMeta(),
	}

	labelSelector := map[string]string{
		"cluster": clusterName,
		"type":    "cassandra-node",
		"state":   "serving",
	}

	if appName, ok := clusterLabels["app"]; ok {
		labelSelector["app"] = appName
	}

	listOpts := &metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelSelector).String(),
	}

	err := c.listerUpdater.List(namespace, pods, sdk.WithListOptions(listOpts))
	if err != nil {
		return nil, fmt.Errorf("Could not list pods for cluster %s: %s", clusterName, err)
	}

	return pods, nil
}
