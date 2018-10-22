package controller

import (
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"github.com/sirupsen/logrus"
)

// reconcile brings the cassandra cluster in kube to the specified state
func (c *ClusterController) reconcile() error {
	saName, err := c.convergeServiceAccount()
	if err != nil {
		return err
	}

	err = c.convergeServices()
	if err != nil {
		return err
	}
	err = c.convergeStatefulSet(saName)
	if err != nil {
		return err
	}

	if c.cluster.Spec.Repair != nil {
		err = c.convergeRepairCronJob()
		if err != nil {
			return err
		}
	}

	if c.cluster.Spec.EnablePodDisruptionBudget {
		err = c.convergeDisruptionBudget()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ClusterController) convergeDisruptionBudget() error {
	logrus.Debugln("Converging PodDisruptionBudget")
	_, err := resource.NewPodDisruptionBudget(c.cluster).Reconcile(c.driver)
	return err
}

func (c *ClusterController) convergeRepairCronJob() error {
	logrus.Debugln("Converging repair cron job")
	_, err := resource.NewRepairCronJob(c.cluster).Reconcile(c.driver)
	return err
}

func (c *ClusterController) convergeStatefulSet(serviceAccountName string) error {
	logrus.Debugln("Converging statefulset")

	_, err := resource.NewStatefulSet(
		c.cluster,
		resource.WithServiceName(c.headlessServiceName),
		resource.WithServiceAccountName(serviceAccountName),
	).Reconcile(c.driver)

	return err
}

func (c *ClusterController) convergeServiceAccount() (string, error) {
	logrus.Debugln("Converging ServiceAccount")
	obj, err := resource.NewServiceAccount(c.cluster).Reconcile(c.driver)
	if err != nil {
		return "", err
	}
	name, _, err := k8sutil.GetNameAndNamespace(obj)

	return name, err
}

// convergeService creates or updates the services required by the operator
func (c *ClusterController) convergeServices() error {
	logrus.Debugln("Converging public service")
	_, err := resource.NewService(c.cluster, resource.WithServiceType(resource.ServiceTypePublicLB)).Reconcile(c.driver)
	if err != nil {
		return err
	}
	logrus.Debugln("Converging internal service")
	_, err = resource.NewService(c.cluster, resource.WithServiceType(resource.ServiceTypeInternal)).Reconcile(c.driver)
	if err != nil {
		return err
	}
	logrus.Debugln("Converging headless service")
	headlessService, err := resource.NewService(c.cluster, resource.WithServiceType(resource.ServiceTypeHeadless)).Reconcile(c.driver)
	if err != nil {
		return err
	}

	c.headlessServiceName, _, err = k8sutil.GetNameAndNamespace(headlessService)
	if err != nil {
		return err
	}

	if c.cluster.Spec.EnablePublicPodServices {
		logrus.Debugln("Converging public pod services")
		for i := 0; i < c.cluster.Spec.Size; i++ {
			_, err = resource.NewService(
				c.cluster,
				resource.WithServiceType(resource.ServiceTypePublicPod),
				resource.WithPodNumber(i),
			).Reconcile(c.driver)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
