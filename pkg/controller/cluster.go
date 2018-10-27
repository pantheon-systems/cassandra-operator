package controller

import (
	"context"
	"fmt"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/version"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ClusterController is the director for they sync and build
type ClusterController struct {
	driver  client.Client
	cluster *v1alpha1.CassandraCluster

	headlessServiceName string
}

// New constructs a new ClusterController from an API object
func New(cc *v1alpha1.CassandraCluster, driver client.Client) *ClusterController {
	return &ClusterController{
		driver:  driver,
		cluster: cc,
	}
}

// Sync syncs the current state to desired
// TODO: Make this able to roll back on error state on first creation
func (c *ClusterController) Sync() (reconcile.Result, error) {
	logrus.Debugln("Sync called")

	switch c.cluster.Status.Phase {
	case "":
		c.cluster.Annotations["database.panth.io/cassandra-operator-version"] = version.Version
		err := c.driver.Update(context.TODO(), c.cluster)
		if err != nil {
			return reconcile.Result{}, err
		}
	case v1alpha1.ClusterPhaseInitial:
		logrus.Debugf("ClusterPhaseInitial for cluster: %s", c.cluster.GetName())
		// initial is the default phase when the cluster object is created
		err := c.validateConfigmaps()
		if err != nil {
			return reconcile.Result{}, err
		}
		// Vault -> maybe a plugin interface here for the OSS project
		err = c.validateSecrets()
		if err != nil {
			return reconcile.Result{}, err
		}
	case v1alpha1.ClusterPhaseFailed:
		return fmt.Errorf("provisioning cluster has failed")
	default:
		if c.cluster.Status.Provisioning() {
			logrus.Debugf("Nodes are provisioning for cluster %s, no-op and wait", c.cluster.GetName())
			return reconcile.Result{RequeueAfter: reconcilePeriod}, nil
		}

		if c.cluster.Status.NodesInTransit() {
			logrus.Debugf("Nodes are in motion for cluster %s, no-op and wait", c.cluster.GetName())
			return reconcile.Result{RequeueAfter: reconcilePeriod}, nil
		}
	}

	return c.reconcile()
}

func (c *ClusterController) validateSecrets() error {
	return nil
}

func (c *ClusterController) validateConfigmaps() error {
	return nil
}
