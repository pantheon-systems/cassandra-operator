package controller_test

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"github.com/pantheon-systems/cassandra-operator/pkg/controller"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// Mock Objects
type MockClusterClient struct {
	GetStatusCallback func(node *corev1.Pod) (map[string]*nodetool.Status, error)
	GetHostIDCallback func(node *corev1.Pod) (string, error)
}

// GetNodeStatus retrieves the specified nodes status
func (c *MockClusterClient) GetStatus(node *corev1.Pod) (map[string]*nodetool.Status, error) {
	return c.GetStatusCallback(node)
}

func (c *MockClusterClient) GetHostID(node *corev1.Pod) (string, error) {
	return c.GetHostIDCallback(node)
}

// Unit Tests
func TestUpdate_NoPods(t *testing.T) {
	// Phase: ClusterPhaseInitial, No Pods
	// deleted: 0
	// creating: 0
	// ready: 0
	// joining: 0
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseInitial
	mockClusterClient := &MockClusterClient{}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(1, v1alpha1.ClusterPhaseInitial)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseInitial, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Ready, 0)
	assert.Len(t, status.Members.Unready, 0)
}

func TestGetClusterStatus_CreatingPodPending(t *testing.T) {
	// Phase: ClusterPhaseCreating, PodPhase: PodPending
	// deleted: 0
	// creating: 1
	// ready: 0
	// joining: 0
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseCreating
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	mockClusterClient := &MockClusterClient{}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(1, v1alpha1.ClusterPhaseCreating)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseCreating, status.Phase)
	assert.Len(t, status.Members.Creating, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Creating[0])
}

func TestGetClusterStatus_CreatingPodRunningNotReady(t *testing.T) {
	// Phase: ClusterPhaseCreating, PodPhase: PodRunning, PodConditionReady: False
	// deleted: 0
	// creating: 0
	// ready: 0
	// joining: 1
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseInitializing
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
	mockClusterClient := &MockClusterClient{}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}

	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(1, v1alpha1.ClusterPhaseCreating)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseCreating, status.Phase)
	assert.Len(t, status.Members.Creating, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Creating[0])
}

func TestGetClusterStatus_CreatingPodRunningReadyJoining(t *testing.T) {
	// Phase: ClusterPhaseInitializing, PodPhase: PodRunning, PodConditionReady: True, Joining
	// deleted: 0
	// creating: 0
	// ready: 0
	// joining: 1
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseInitializing
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return map[string]*nodetool.Status{
				"4d1a5c32-9642-405e-bd7e-27c8400bf779": {
					HostID: "4d1a5c32-9642-405e-bd7e-27c8400bf779",
					Owns:   100.0,
					State:  nodetool.NodeStateJoining,
					Status: nodetool.NodeStatusUp,
				},
			}, nil
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			return "4d1a5c32-9642-405e-bd7e-27c8400bf779", nil
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(1, v1alpha1.ClusterPhaseInitializing)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseInitializing, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Joining[0])
}

func TestGetClusterStatus_CreatingOtherUnsupported(t *testing.T) {
	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return map[string]*nodetool.Status{
				"4d1a5c32-9642-405e-bd7e-27c8400bf779": {
					HostID: "4d1a5c32-9642-405e-bd7e-27c8400bf779",
					Owns:   100.0,
					State:  nodetool.NodeStateNormal,
					Status: nodetool.NodeStatusUp,
				},
			}, nil
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			return "4d1a5c32-9642-405e-bd7e-27c8400bf779", nil
		},
	}

	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-cluster-cassandra-0",
							Namespace: "testnamespace",
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodSucceeded,
						},
					},
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(1, v1alpha1.ClusterPhaseCreating)
	err := controller.Update(cc)

	assert.Error(t, err, "Unsupported PodPhase: Succeeded")
}

func TestGetClusterStatus_AllRunningUp(t *testing.T) {
	// Phase: ClusterPhaseRunning, PodPhase: PodRunning, PodConditionReady: True, Up/Normal
	// deleted: 0
	// creating: 0
	// ready: 1
	// joining: 0
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseRunning
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return map[string]*nodetool.Status{
				"4d1a5c32-9642-405e-bd7e-27c8400bf779": {
					HostID: "4d1a5c32-9642-405e-bd7e-27c8400bf779",
					Owns:   100.0,
					State:  nodetool.NodeStateNormal,
					Status: nodetool.NodeStatusUp,
				},
			}, nil
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			return "4d1a5c32-9642-405e-bd7e-27c8400bf779", nil
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(1, v1alpha1.ClusterPhaseRunning)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseRunning, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])
}

func TestGetClusterStatus_ScalingRunningJoin(t *testing.T) {
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	mockPod2 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-1",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
					mockPod2,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}

	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return map[string]*nodetool.Status{
				"4d1a5c32-9642-405e-bd7e-27c8400bf779": {
					HostID: "4d1a5c32-9642-405e-bd7e-27c8400bf779",
					Owns:   100.0,
					State:  nodetool.NodeStateNormal,
					Status: nodetool.NodeStatusUp,
				},
				"f652fd91-3c7d-43bb-84a2-b6d55e578b49": {
					HostID: "f652fd91-3c7d-43bb-84a2-b6d55e578b49",
					Owns:   100.0,
					State:  nodetool.NodeStateJoining,
					Status: nodetool.NodeStatusUp,
				},
			}, nil
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			if node.GetName() == "test-cluster-cassandra-0" {
				return "4d1a5c32-9642-405e-bd7e-27c8400bf779", nil
			}
			if node.GetName() == "test-cluster-cassandra-1" {
				return "f652fd91-3c7d-43bb-84a2-b6d55e578b49", nil
			}

			return "", fmt.Errorf("SHOULD NEVER GET HERE")
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	// Phase: ClusterPhaseRunning, PodPhase: PodRunning/PodRunning, PodConditionReady: True/True, Up-Normal/Up-Joining
	// deleted: 0
	// creating: 0
	// ready: 1
	// joining: 1
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseScaling
	cc := getCassandraCluster(2, v1alpha1.ClusterPhaseRunning)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseScaling, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 1)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])

	// Phase: ClusterPhaseScaling, PodPhase: PodRunning/PodRunning, PodConditionReady: True/True, Up-Normal/Up-Joining
	// deleted: 0
	// creating: 0
	// ready: 1
	// joining: 1
	// leaving: 0
	// unready: 0
	// NewPhase: ClusterPhaseScaling
	cc = getCassandraCluster(2, v1alpha1.ClusterPhaseScaling)
	err = controller.Update(cc)
	status = capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseScaling, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 1)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])
	assert.Equal(t, mockPod2.GetName(), status.Members.Joining[0])
}

func TestGetClusterStatus_ScalingRunningLeave(t *testing.T) {
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	mockPod2 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-1",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
					mockPod2,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}

	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return map[string]*nodetool.Status{
				"4d1a5c32-9642-405e-bd7e-27c8400bf779": {
					HostID: "4d1a5c32-9642-405e-bd7e-27c8400bf779",
					Owns:   100.0,
					State:  nodetool.NodeStateNormal,
					Status: nodetool.NodeStatusUp,
				},
				"f652fd91-3c7d-43bb-84a2-b6d55e578b49": {
					HostID: "f652fd91-3c7d-43bb-84a2-b6d55e578b49",
					Owns:   100.0,
					State:  nodetool.NodeStateLeaving,
					Status: nodetool.NodeStatusUp,
				},
			}, nil
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			if node.GetName() == "test-cluster-cassandra-0" {
				return "4d1a5c32-9642-405e-bd7e-27c8400bf779", nil
			}
			if node.GetName() == "test-cluster-cassandra-1" {
				return "f652fd91-3c7d-43bb-84a2-b6d55e578b49", nil
			}

			return "", fmt.Errorf("SHOULD NEVER GET HERE")
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	// Phase: ClusterPhaseScaling, PodPhase: PodRunning/PodRunning, PodConditionReady: True/True, Up-Normal/Up-Joining
	// deleted: 0
	// creating: 0
	// ready: 1
	// joining: 0
	// leaving: 1
	// unready: 0
	// NewPhase: ClusterPhaseScaling
	cc := getCassandraCluster(2, v1alpha1.ClusterPhaseScaling)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseScaling, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 1)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])
	assert.Equal(t, mockPod2.GetName(), status.Members.Leaving[0])

	// Phase: ClusterPhaseRunning, PodPhase: PodRunning/PodRunning, PodConditionReady: True/True, Up-Normal/Up-Joining
	// deleted: 0
	// creating: 0
	// ready: 1
	// joining: 0
	// leaving: 1
	// unready: 0
	// NewPhase: ClusterPhaseScaling
	cc = getCassandraCluster(2, v1alpha1.ClusterPhaseRunning)
	err = controller.Update(cc)
	status = capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseScaling, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 1)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])
	assert.Equal(t, mockPod2.GetName(), status.Members.Leaving[0])
}

func TestGetClusterStatus_ScalingRunningCreating(t *testing.T) {
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	mockPod2 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-1",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
					mockPod2,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}

	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return map[string]*nodetool.Status{
				"4d1a5c32-9642-405e-bd7e-27c8400bf779": {
					HostID: "4d1a5c32-9642-405e-bd7e-27c8400bf779",
					Owns:   100.0,
					State:  nodetool.NodeStateNormal,
					Status: nodetool.NodeStatusUp,
				},
			}, nil
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			return "4d1a5c32-9642-405e-bd7e-27c8400bf779", nil
		},
	}
	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	// Phase: ClusterPhaseScaling, PodPhase: PodRunning/PodRunning, PodConditionReady: True/True, Up-Normal/Up-Joining
	// deleted: 0
	// creating: 1
	// ready: 1
	// joining: 0
	// leaving: 0
	// unready: 0
	cc := getCassandraCluster(2, v1alpha1.ClusterPhaseScaling)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseScaling, status.Phase)
	assert.Len(t, status.Members.Creating, 1)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])
	assert.Equal(t, mockPod2.GetName(), status.Members.Creating[0])

	// Phase: ClusterPhaseRunning, PodPhase: PodRunning/PodRunning, PodConditionReady: True/True, Up-Normal/Up-Joining
	// deleted: 0
	// creating: 1
	// ready: 1
	// joining: 0
	// leaving: 0
	// unready: 0
	cc = getCassandraCluster(2, v1alpha1.ClusterPhaseRunning)
	err = controller.Update(cc)
	status = capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseScaling, status.Phase)
	assert.Len(t, status.Members.Creating, 1)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 0)
	assert.Len(t, status.Members.Ready, 1)
	assert.Equal(t, mockPod1.GetName(), status.Members.Ready[0])
	assert.Equal(t, mockPod2.GetName(), status.Members.Creating[0])
}

func TestGetClusterStatus_CreatingFailure(t *testing.T) {
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}

	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return nil, fmt.Errorf("SHOULD NOT GET HERE")
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			return "nil", fmt.Errorf("SHOULD NOT GET HERE")
		},
	}

	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(2, v1alpha1.ClusterPhaseCreating)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseFailed, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 1)
	assert.Len(t, status.Members.Ready, 0)

	cc = getCassandraCluster(2, v1alpha1.ClusterPhaseInitializing)
	err = controller.Update(cc)
	status = capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseFailed, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 1)
	assert.Len(t, status.Members.Ready, 0)
}

func TestGetClusterStatus_CreatingUnknown(t *testing.T) {
	mockPod1 := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-cassandra-0",
			Namespace: "testnamespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodUnknown,
		},
	}

	var capturedObject *v1alpha1.CassandraCluster
	mockKubeClient := &k8s.MockClient{
		ListCallback: func(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
			actual := &corev1.PodList{
				Items: []corev1.Pod{
					mockPod1,
				},
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(actual, into)
		},
		UpdateCallback: func(object sdk.Object) error {
			capturedObject = object.(*v1alpha1.CassandraCluster)
			return nil
		},
	}

	mockClusterClient := &MockClusterClient{
		GetStatusCallback: func(node *corev1.Pod) (map[string]*nodetool.Status, error) {
			return nil, fmt.Errorf("SHOULD NOT GET HERE")
		},
		GetHostIDCallback: func(node *corev1.Pod) (string, error) {
			return "nil", fmt.Errorf("SHOULD NOT GET HERE")
		},
	}

	controller := controller.NewStatusManager(mockClusterClient, mockKubeClient)

	cc := getCassandraCluster(2, v1alpha1.ClusterPhaseCreating)
	err := controller.Update(cc)
	status := capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseFailed, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 1)
	assert.Len(t, status.Members.Ready, 0)

	cc = getCassandraCluster(2, v1alpha1.ClusterPhaseInitializing)
	err = controller.Update(cc)
	status = capturedObject.Status

	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.ClusterPhaseFailed, status.Phase)
	assert.Len(t, status.Members.Creating, 0)
	assert.Len(t, status.Members.Joining, 0)
	assert.Len(t, status.Members.Leaving, 0)
	assert.Len(t, status.Members.Unready, 1)
	assert.Len(t, status.Members.Ready, 0)
}

func getCassandraCluster(size int, phase v1alpha1.ClusterPhase) *v1alpha1.CassandraCluster {
	return &v1alpha1.CassandraCluster{
		Status: v1alpha1.ClusterStatus{
			Phase: phase,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "testnamespace",
		},
		Spec: v1alpha1.ClusterSpec{
			Size: size,
		},
	}
}
