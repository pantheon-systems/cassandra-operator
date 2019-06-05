package cassandracluster

import (
	"github.com/Sirupsen/logrus"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// reconcile brings the cassandra cluster in kube to the specified state
func (c *ReconcileCassandraCluster) reconcile(cluster *v1alpha1.CassandraCluster) error {
	logrus.Debugln("Converging ServiceAccount")
	serviceAccount, err := resource.NewServiceAccount(cluster).Reconcile(c.ctx, c.client)
	if err != nil {
		return err
	}

	saMetaAccessor, err := meta.Accessor(serviceAccount)
	if err != nil {
		return err
	}

	logrus.Debugln("Converging public service")
	_, err = resource.NewService(cluster, resource.WithServiceType(resource.ServiceTypePublicLB)).Reconcile(c.ctx, c.client)
	if err != nil {
		return err
	}

	logrus.Debugln("Converging internal service")
	_, err = resource.NewService(cluster, resource.WithServiceType(resource.ServiceTypeInternal)).Reconcile(c.ctx, c.client)
	if err != nil {
		return err
	}

	logrus.Debugln("Converging headless service")
	headless, err := resource.NewService(cluster, resource.WithServiceType(resource.ServiceTypeHeadless)).Reconcile(c.ctx, c.client)
	if err != nil {
		return err
	}

	if cluster.Spec.EnablePublicPodServices {
		_, err := c.convergePublicPodServices(cluster)
		if err != nil {
			return err
		}
	}

	headlessMetaAccessor, err := meta.Accessor(headless)
	if err != nil {
		return err
	}

	logrus.Debugln("Converging statefulset")
	_, err = resource.NewStatefulSet(
		cluster,
		resource.WithServiceName(headlessMetaAccessor.GetName()),
		resource.WithServiceAccountName(saMetaAccessor.GetName()),
	).Reconcile(c.ctx, c.client)
	if err != nil {
		return err
	}

	if cluster.Spec.Repair != nil {
		logrus.Debugln("Converging repair cron job")
		_, err := resource.NewRepairCronJob(cluster).Reconcile(c.ctx, c.client)
		if err != nil {
			return err
		}
	}

	if cluster.Spec.EnablePodDisruptionBudget {
		logrus.Debugln("Converging PodDisruptionBudget")
		_, err := resource.NewPodDisruptionBudget(cluster).Reconcile(c.ctx, c.client)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ReconcileCassandraCluster) convergePublicPodServices(cluster *v1alpha1.CassandraCluster) ([]runtime.Object, error) {
	services := []runtime.Object{}
	logrus.Debugln("Converging public pod services")
	for i := 0; i < cluster.Spec.Size; i++ {
		podPublic, err := resource.NewService(
			cluster,
			resource.WithServiceType(resource.ServiceTypePublicPod),
			resource.WithPodNumber(i),
		).Reconcile(c.ctx, c.client)
		if err != nil {
			return nil, err
		}
		services[i] = podPublic
	}

	return services, nil
}
