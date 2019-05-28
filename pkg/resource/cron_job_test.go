package resource_test

import (
	"errors"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	three   int32 = 3
	zero    int32 = 0
	trueVar       = true
)

func TestRepairCronJob_Reconcile(t *testing.T) {
	type fields struct {
		cluster *v1alpha1.CassandraCluster
		actual  *batchv1beta1.CronJob
	}
	tests := []struct {
		name    string
		fields  fields
		want    sdk.Object
		wantErr bool
	}{
		{
			name: "clean-state",
			fields: fields{
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Spec: v1alpha1.ClusterSpec{
						Repair: &v1alpha1.RepairPolicy{
							Schedule: "",
							Image:    "",
						},
					},
				},
				actual: nil,
			},
			want: &batchv1beta1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batch/v1beta1",
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra-repair",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster": "test-cluster-1",
						"app":     "test-app",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
					},
				},
				Spec: batchv1beta1.CronJobSpec{
					Schedule:                   "",
					ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
					SuccessfulJobsHistoryLimit: &three,
					FailedJobsHistoryLimit:     &three,
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: &zero,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "test-cluster-1-cassandra-repair",
											Image: "",
											Env: []corev1.EnvVar{
												{
													Name:  "CASSANDRA_CLUSTER",
													Value: "test-cluster-1",
												},
												{
													Name: "KUBE_NAMESPACE",
													ValueFrom: &corev1.EnvVarSource{
														FieldRef: &corev1.ObjectFieldSelector{
															FieldPath: "metadata.namespace",
														},
													},
												},
												{
													Name:  "APP_NAME",
													Value: "test-app",
												},
											},
										},
									},
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "previous-value",
			fields: fields{
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Spec: v1alpha1.ClusterSpec{
						Repair: &v1alpha1.RepairPolicy{
							Schedule: "",
							Image:    "",
						},
					},
				},
				actual: &batchv1beta1.CronJob{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "batch/v1beta1",
						Kind:       "CronJob",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1-cassandra-repair",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"cluster": "test-cluster-1",
							"app":     "test-app",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-cluster-1",
								Controller: &trueVar,
							},
						},
						ResourceVersion: "test-resource-version-update",
					},
				},
			},
			want: &batchv1beta1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batch/v1beta1",
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra-repair",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster": "test-cluster-1",
						"app":     "test-app",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
					},
					ResourceVersion: "test-resource-version-update",
				},
				Spec: batchv1beta1.CronJobSpec{
					Schedule:                   "",
					ConcurrencyPolicy:          batchv1beta1.ForbidConcurrent,
					SuccessfulJobsHistoryLimit: &three,
					FailedJobsHistoryLimit:     &three,
					JobTemplate: batchv1beta1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							BackoffLimit: &zero,
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "test-cluster-1-cassandra-repair",
											Image: "",
											Env: []corev1.EnvVar{
												{
													Name:  "CASSANDRA_CLUSTER",
													Value: "test-cluster-1",
												},
												{
													Name: "KUBE_NAMESPACE",
													ValueFrom: &corev1.EnvVarSource{
														FieldRef: &corev1.ObjectFieldSelector{
															FieldPath: "metadata.namespace",
														},
													},
												},
												{
													Name:  "APP_NAME",
													Value: "test-app",
												},
											},
										},
									},
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "fail-to-get",
			fields: fields{
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
					Spec: v1alpha1.ClusterSpec{
						Repair: &v1alpha1.RepairPolicy{
							Schedule: "",
							Image:    "",
						},
					},
				},
				actual: nil,
			},
			want:    nil,
			wantErr: true,
		},
	}
	callCount := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubeClient := &k8s.MockClient{
				GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
					callCount++
					switch callCount {
					case 2:
						if err := k8sutil.RuntimeObjectIntoRuntimeObject(tt.fields.actual, into); err != nil {
							return err
						}
						return nil
					case 3:
						return errors.New("some error")
					}

					return nil
				},
			}

			b := resource.NewRepairCronJob(tt.fields.cluster)
			got, err := b.Reconcile(mockKubeClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("RepairCronJob.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RepairCronJob.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
