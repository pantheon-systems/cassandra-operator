package controller

import (
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	podFinalizer = "finalizer.cassandra.database.pantheon.io/v1alpha1"
)

// PodFinalizerController is the controller for cassandra node (pod) finalizers
type PodFinalizerController struct {
	k8sDriver        k8s.Client
	finalizerManager *k8s.Finalizer
	nodetoolDriver   *nodetool.Executor
}

// NewPodFinalizerController builds a new PodFinalizerController
func NewPodFinalizerController(k8sDriver k8s.Client, nodetoolDriver *nodetool.Executor) *PodFinalizerController {
	return &PodFinalizerController{
		k8sDriver:        k8sDriver,
		finalizerManager: k8s.NewFinalizer(k8sDriver, podFinalizer),
		nodetoolDriver:   nodetoolDriver,
	}
}

// Converge takes a node verifies it is supposed to have the finalizer and
// adds it if not already there
func (c *PodFinalizerController) Converge(node *corev1.Pod) error {
	// Nodes should have the finalizer. (fast path)
	if c.finalizerManager.NeedToAdd(node) {
		// TODO: We should move this to an MutatingAdmissionWebhook that does
		// not allow the pod to run till the finalizer is added to it.
		return c.finalizerManager.Add(node)
	}

	return nil
}

// Process pod for finalizer
func (c *PodFinalizerController) Process(node *corev1.Pod) error {
	// if its not a delete candidate then we can just bail (fast path)
	if !c.finalizerManager.IsDeletionCandidate(node) {
		return nil
	}

	// get cassandra cluster to get expected size
	cluster := &v1alpha1.CassandraCluster{
		TypeMeta: resource.GetCassandraClusterTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      node.Labels["cluster"],
			Namespace: node.GetNamespace(),
		},
	}
	err := c.k8sDriver.Get(cluster)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if cluster.Status.Provisioning() {
		logrus.Debugf("cluster '%s' is provisioning, cannot change state of node '%s'\n", cluster.GetName(), node.GetName())
		return nil
	}

	err = c.nodetoolDriver.Drain(node)
	if err == nil {
		err = c.nodetoolDriver.Stop(node)
	}

	if err != nil {
		// Drain or decomission failed, we do not proceed
		// with delete
		return err
	}

	// we have successfully drained or decommissioned
	// we can now proceed with the deletion of the pod
	// by kubernetes by removing the finalizer
	return c.finalizerManager.Remove(node)
}
