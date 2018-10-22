package controller_test

import (
	"errors"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"github.com/pantheon-systems/cassandra-operator/pkg/controller"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
	"time"
)

func TestFinalizerController_ProcessNotDeletionCandidate(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: nil,
		},
	}

	mockK8sDriver := k8s.MockClient{}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Process(&testPod)

	assert.NoError(t, err)

	testPod.ObjectMeta.DeletionTimestamp = &now
	err = obj.Process(&testPod)
	assert.NoError(t, err)
}

func TestFinalizerController_ProcessNotPartOfCluster(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
		},
	}

	mockK8sDriver := k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			_, gr := schema.ParseResourceArg("database.pantheon.io/v1alpha1")
			return k8serrors.NewNotFound(gr, "somecluster")
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Process(&testPod)
	assert.NoError(t, err)
}

func TestFinalizerController_ProcessGetClusterError(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
		},
	}

	mockK8sDriver := k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return errors.New("some other error besides not found")
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Process(&testPod)
	assert.Error(t, err)
}

func TestFinalizerController_ProcessStatefulSetGetError(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "statefulsetname",
				},
			},
		},
	}

	cluster := &v1alpha1.CassandraCluster{}

	mockK8sDriver := k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			if into.GetObjectKind().GroupVersionKind().Kind == "CassandraCluster" {
				return k8sutil.RuntimeObjectIntoRuntimeObject(cluster, into)
			}

			return errors.New("some other error besides not found")
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Process(&testPod)
	assert.Error(t, err)
}

func TestFinalizerController_ProcessDrainErrors(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "statefulsetname",
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "cassandra",
				},
			},
		},
	}

	cluster := &v1alpha1.CassandraCluster{
		Spec: v1alpha1.ClusterSpec{
			Size: 3,
		},
	}

	three := int32(3)
	statefulSet := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &three,
		},
	}

	calledDrain := false
	calledUpdate := false

	mockK8sDriver := k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			if into.GetObjectKind().GroupVersionKind().Kind == "CassandraCluster" {
				return k8sutil.RuntimeObjectIntoRuntimeObject(cluster, into)
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(statefulSet, into)
		},
		RunCallback: func(pod *corev1.Pod, containerIdx int, command []string) (string, string, error) {
			if command[1] == "drain" {
				calledDrain = true
			}
			return "", "", errors.New("some big error")
		},
		UpdateCallback: func(object sdk.Object) error {
			calledUpdate = true
			return nil
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Process(&testPod)
	assert.Error(t, err)
	assert.True(t, calledDrain)
	assert.False(t, calledUpdate)
}

func TestFinalizerController_ProcessDrainNoErrors(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "statefulsetname",
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "cassandra",
				},
			},
		},
	}

	cluster := &v1alpha1.CassandraCluster{
		Spec: v1alpha1.ClusterSpec{
			Size: 3,
		},
	}

	three := int32(3)
	statefulSet := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &three,
		},
	}

	calledDrain := false

	mockK8sDriver := k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			if into.GetObjectKind().GroupVersionKind().Kind == "CassandraCluster" {
				return k8sutil.RuntimeObjectIntoRuntimeObject(cluster, into)
			}
			return k8sutil.RuntimeObjectIntoRuntimeObject(statefulSet, into)
		},
		RunCallback: func(pod *corev1.Pod, containerIdx int, command []string) (string, string, error) {
			if command[1] == "drain" {
				calledDrain = true
			}
			return "", "", nil
		},
		UpdateCallback: func(object sdk.Object) error {
			return nil
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Process(&testPod)
	assert.NoError(t, err)
	assert.True(t, calledDrain)
}

// TODO: Complete decommission unit tests
// func TestFinalizerController_ProcessDecommissionNoErrors(t *testing.T) {
// 	testPod := corev1.Pod{
// 		ObjectMeta: metav1.ObjectMeta{
// 			DeletionTimestamp: &metav1.Time{time.Now()},
// 			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
// 			OwnerReferences: []metav1.OwnerReference{
// 				{
// 					Name: "statefulsetname",
// 				},
// 			},
// 		},
// 		Spec: corev1.PodSpec{
// 			Containers: []corev1.Container{
// 				{
// 					Name: "cassandra",
// 				},
// 			},
// 		},
// 	}

// 	cluster := &v1alpha1.CassandraCluster{
// 		Spec: v1alpha1.ClusterSpec{
// 			Size: 2,
// 		},
// 	}

// 	three := int32(3)
// 	statefulSet := &appsv1.StatefulSet{
// 		Spec: appsv1.StatefulSetSpec{
// 			Replicas: &three,
// 		},
// 	}

// 	calledDecom := false

// 	mockK8sDriver := k8s.MockClient{
// 		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
// 			if into.GetObjectKind().GroupVersionKind().Kind == "CassandraCluster" {
// 				return k8sutil.RuntimeObjectIntoRuntimeObject(cluster, into)
// 			}
// 			return k8sutil.RuntimeObjectIntoRuntimeObject(statefulSet, into)
// 		},
// 		RunCallback: func(pod *corev1.Pod, containerIdx int, command []string) (string, string, error) {
// 			if command[1] == "decommission" {
// 				calledDecom = true
// 			}
// 			if command[1] == "info" {
// 				return "ID: 3b920369-cd41-4b6b-8f5f-192f1202ee18", "", nil
// 			}
// 			if command[1] == "status" {
// 				return nodetool.TestStatusOutput, "", nil
// 			}
// 			return "", "", nil
// 		},
// 		UpdateCallback: func(object sdk.Object) error {
// 			return nil
// 		},
// 	}
// 	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

// 	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

// 	err := obj.Process(&testPod)
// 	assert.NoError(t, err)
// 	assert.True(t, calledDecom)
// }

func TestFinalizerController_ConvergeNeedToAddNoError(t *testing.T) {
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: nil,
		},
	}

	mockK8sDriver := k8s.MockClient{
		UpdateCallback: func(object sdk.Object) error {
			return nil
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Converge(&testPod)
	assert.NoError(t, err)

	finalizers := testPod.GetFinalizers()
	assert.Len(t, finalizers, 1)
	assert.Equal(t, finalizers[0], "finalizer.cassandra.database.pantheon.io/v1alpha1")
}

func TestFinalizerController_ConvergeNoNeedToAddAlreadyDeleted(t *testing.T) {
	now := metav1.NewTime(time.Now())
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &now,
		},
	}

	mockK8sDriver := k8s.MockClient{
		UpdateCallback: func(object sdk.Object) error {
			return errors.New("THIS ERROR SHOULD NEVER HAPPEN CAUSE WE SHOULD NOT BE CALLING UPDATE")
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Converge(&testPod)
	assert.NoError(t, err)
}

func TestFinalizerController_ConvergeNoNeedToAdd(t *testing.T) {
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: nil,
			Finalizers:        []string{"finalizer.cassandra.database.pantheon.io/v1alpha1"},
		},
	}

	mockK8sDriver := k8s.MockClient{
		UpdateCallback: func(object sdk.Object) error {
			return errors.New("THIS ERROR SHOULD NEVER HAPPEN CAUSE WE SHOULD NOT BE CALLING UPDATE")
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Converge(&testPod)
	assert.NoError(t, err)
}

func TestFinalizerController_ConvergeNeedToAddError(t *testing.T) {
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: nil,
		},
	}

	mockK8sDriver := k8s.MockClient{
		UpdateCallback: func(object sdk.Object) error {
			return errors.New("some error")
		},
	}
	nodetoolDriver := nodetool.NewExecutor(&mockK8sDriver)

	obj := controller.NewPodFinalizerController(&mockK8sDriver, nodetoolDriver)

	err := obj.Converge(&testPod)
	assert.Error(t, err)
}
