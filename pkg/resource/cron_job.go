package resource

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	v1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var (
	successfulJobsHistoryLimit int32 = 3
	failedJobsHistoryLimit     int32 = 3
	backoffLimit               int32 // default 0
)

// RepairCronJob class that builds a Repair Cron Job for the cassandra cluster
type RepairCronJob struct {
	cluster *v1alpha1.CassandraCluster
	desired *batchv1beta1.CronJob
}

// NewRepairCronJob constructor for RepairCronJob
func NewRepairCronJob(cc *v1alpha1.CassandraCluster) *RepairCronJob {
	return &RepairCronJob{
		cluster: cc,
	}
}

// Reconcile the cron job's actual state with desired
func (b *RepairCronJob) Reconcile(ctx context.Context, driver client.Client) (runtime.Object, error) {
	var err error

	b.configureDesired()

	namespacedName := types.NamespacedName{
		Namespace: b.desired.GetNamespace(),
		Name:      b.desired.GetName(),
	}

	existing := &batchv1beta1.CronJob{}
	err = driver.Get(ctx, namespacedName, existing)
	if err != nil {
		return nil, errors.New("could not get existing")
	}

	if existing.ResourceVersion != "" {
		// here we have one that is existing and one that is expected
		// we put our code here to reconcile the two and return
		// the reconciled object
		b.desired.ResourceVersion = existing.ResourceVersion
		err = driver.Update(ctx, b.desired)
		return b.desired, err
	}

	err = driver.Create(ctx, b.desired)
	return b.desired, err
}

// Build creates the object in preparation to save
func (b *RepairCronJob) configureDesired() {
	b.desired = &batchv1beta1.CronJob{
		TypeMeta: GetCronJobTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.buildCronJobName(),
			Namespace: b.cluster.GetNamespace(),
			Labels:    b.buildLabels(),
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule:                   b.cluster.Spec.Repair.Schedule,
			ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &successfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     &failedJobsHistoryLimit,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: v1.JobSpec{
					BackoffLimit: &backoffLimit,
					Template:     b.buildCronJobPodTemplateSpec(),
				},
			},
		},
	}
	//b.setOwner(asOwner(b.cluster))
}

func (b *RepairCronJob) buildCronJobName() string {
	return fmt.Sprintf(cronJobNameTemplate, b.cluster.GetName())
}

func (b *RepairCronJob) buildLabels() map[string]string {
	labels := map[string]string{
		"cluster": b.cluster.GetName(),
	}

	if appName, ok := b.cluster.ObjectMeta.Labels["app"]; ok {
		labels["app"] = appName
	}

	return labels
}

func (b *RepairCronJob) buildCronJobPodTemplateSpec() corev1.PodTemplateSpec {
	imageName := b.cluster.Spec.Repair.Image
	return corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  b.buildCronJobName(),
					Image: imageName,
					Env:   b.buildCronJobEnvVars(),
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func (b *RepairCronJob) buildCronJobEnvVars() []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  cassandraClusterEnvVar,
			Value: b.cluster.GetName(),
		},
		{
			Name: kubeNamespaceEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}

	if appName, ok := b.cluster.ObjectMeta.Labels["app"]; ok {
		envs = append(envs, corev1.EnvVar{
			Name:  appNameEnvVar,
			Value: appName,
		})
	}

	return envs
}

// func (b *RepairCronJob) setOwner(owner metav1.OwnerReference) {
// 	b.desired.SetOwnerReferences(append(b.desired.GetOwnerReferences(), owner))
// }
