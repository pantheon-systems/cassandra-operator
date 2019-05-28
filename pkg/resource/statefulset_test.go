package resource_test

import (
	"errors"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kuberesource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	one      int32  = 1
	two      int32  = 2
	fifteen  int32  = 15
	five     int32  = 5
	ssd      string = "ssd"
	capacity        = kuberesource.MustParse("1000Gi")
)

func TestStatefulSet_Reconcile(t *testing.T) {
	type fields struct {
		actual  *appsv1.StatefulSet
		cluster *v1alpha1.CassandraCluster
		options []resource.BuilderOption
	}
	tests := []struct {
		name      string
		fields    fields
		want      sdk.Object
		wantErr   bool
		getErr    error
		updateErr error
	}{
		{
			name: "no-service-account-name",
			fields: fields{
				actual:  nil,
				cluster: &v1alpha1.CassandraCluster{},
				options: []resource.BuilderOption{
					resource.WithServiceName("some-service-name"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no-service-name",
			fields: fields{
				actual:  nil,
				cluster: &v1alpha1.CassandraCluster{},
				options: []resource.BuilderOption{
					resource.WithServiceAccountName("some-service-account-name"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "get-error",
			fields: fields{
				actual:  nil,
				cluster: &v1alpha1.CassandraCluster{},
				options: []resource.BuilderOption{
					resource.WithServiceAccountName("some-service-account-name"),
					resource.WithServiceName("some-service-name"),
				},
			},
			want:    nil,
			wantErr: true,
			getErr:  errors.New("some error"),
		},
		{
			name: "update-error",
			fields: fields{
				actual: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "some-resource-version",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &two,
					},
					Status: appsv1.StatefulSetStatus{
						ReadyReplicas: 2,
					},
				},
				cluster: getBaseInputCluster(),
				options: []resource.BuilderOption{
					resource.WithServiceAccountName("some-service-account-name"),
					resource.WithServiceName("some-service-name"),
				},
			},
			want:      nil,
			wantErr:   true,
			updateErr: errors.New("some error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &k8s.MockClient{
				GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
					if tt.getErr != nil {
						return tt.getErr
					}

					if tt.fields.actual != nil {
						if err := k8sutil.RuntimeObjectIntoRuntimeObject(tt.fields.actual, into); err != nil {
							return err
						}
					}

					return nil
				},
				UpdateCallback: func(object sdk.Object) error {
					if tt.updateErr != nil {
						return tt.updateErr
					}

					return nil
				},
			}
			b := resource.NewStatefulSet(tt.fields.cluster, tt.fields.options...)
			got, err := b.Reconcile(mockClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("StatefulSet.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatefulSet_ReconcileDefaults(t *testing.T) {
	cluster := getBaseInputCluster()
	expected := getBaseExpectedStatefulSet()

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileStorageClass(t *testing.T) {
	storageClassName := "not-default"

	cluster := getBaseInputCluster()
	cluster.Spec.Node.PersistentVolume = &v1alpha1.PersistentVolumeSpec{
		StorageClassName: storageClassName,
	}

	expected := getBaseExpectedStatefulSet()
	expected.Spec.VolumeClaimTemplates[0].ObjectMeta.Annotations = map[string]string{
		"volume.beta.kubernetes.io/storage-class": storageClassName,
	}
	expected.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = &storageClassName

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileJvmAgent(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.JvmAgent = "sidecar"

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Template.ObjectMeta.Annotations = map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "9126",
	}

	telegrafContainer := corev1.Container{
		Name:            "telegraf",
		Image:           "telegraf:1.2",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			"--config",
			"/telegraf-config/telegraf.conf",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 9126,
				Name:          "prometheus",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "jvm-agent-config",
				MountPath: "/telegraf-config",
			},
			{
				Name:      "test-cluster-1-cassandra-data",
				MountPath: "/var/lib/cassandra",
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    kuberesource.MustParse("1"),
				corev1.ResourceMemory: kuberesource.MustParse("128Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    kuberesource.MustParse("0.1"),
				corev1.ResourceMemory: kuberesource.MustParse("64Mi"),
			},
		},
	}

	expected.Spec.Template.Spec.Containers = append(expected.Spec.Template.Spec.Containers, telegrafContainer)

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}

	cluster.Spec.JvmAgent = "jvm"
	expected = getBaseExpectedStatefulSet()
	expected.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		expected.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{
			Name:      "jvm-agent-config",
			MountPath: "/jvm-agent",
		},
	)

	statefulset = getNewSS(cluster)
	got, err = statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAffinityAndAnti(t *testing.T) {
	affinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"cluster": "test-cluster-1",
							"app":     "test-app",
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}

	cluster := getBaseInputCluster()
	cluster.Spec.Affinity = affinity

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Template.Spec.Affinity = affinity

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileSecretAndConfigNames(t *testing.T) {
	jvmAgentConfigName := "some-other-config-map-name"

	cluster := getBaseInputCluster()
	cluster.Spec.JvmAgentConfigName = jvmAgentConfigName

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Template.Spec.Volumes[1].VolumeSource.ConfigMap.LocalObjectReference.Name = jvmAgentConfigName

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}

	secretName := "some-other-secret-name"

	cluster = getBaseInputCluster()
	cluster.Spec.SecretName = secretName

	expected = getBaseExpectedStatefulSet()
	expected.Spec.Template.Spec.Volumes[0].VolumeSource.Secret.SecretName = secretName

	statefulset = getNewSS(cluster)
	got, err = statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize2Replica1Ready1(t *testing.T) {
	cluster := getBaseInputCluster()

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &one
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = one

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &two
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-1.some-service-name.test-namespace.svc.cluster.local"
	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "true"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize1Replica2Ready2(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.Size = 1

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &two
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = two

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &one
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize1Replica3Ready3(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.Size = 1

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &three
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = three

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &two
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-1.some-service-name.test-namespace.svc.cluster.local"
	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "true"
	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize3Replica1Ready1(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.Size = 3

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &one
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = one

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &two
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-1.some-service-name.test-namespace.svc.cluster.local"
	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "true"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize3Replica2Ready2(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.Size = 3

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &two
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = two

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &three
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-1.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-2.some-service-name.test-namespace.svc.cluster.local"
	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "true"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize2Replica1Ready0(t *testing.T) {
	cluster := getBaseInputCluster()

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &one
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = zero

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &one
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize1Replica1Ready1(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.Size = 1

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &one
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = one

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &one
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileAlreadyExistsSize2Replica2Ready2(t *testing.T) {
	cluster := getBaseInputCluster()

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &two
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = two

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &two
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-1.some-service-name.test-namespace.svc.cluster.local"
	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "true"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}
func TestStatefulSet_ReconcileAlreadyExistsSize2Replica0Ready0(t *testing.T) {
	cluster := getBaseInputCluster()

	existing := getBaseExpectedStatefulSet()
	existing.Spec.Replicas = &zero
	existing.ObjectMeta.ResourceVersion = "some-resource-version"
	existing.Status.ReadyReplicas = zero

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Replicas = &one
	expected.ObjectMeta.ResourceVersion = "some-resource-version"
	expected.Spec.Template.Spec.Containers[0].Env[7].Value = "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local"
	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "false"

	mockClient := &k8s.MockClient{
		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
		},
	}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileDatacenter(t *testing.T) {
	datacenterName := "some-other-dc"

	cluster := getBaseInputCluster()
	cluster.Spec.Datacenter = datacenterName

	expected := getBaseExpectedStatefulSet()
	expected.Spec.Template.Spec.Containers[0].Env = append(
		expected.Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{
			Name:  "CASSANDRA_DC",
			Value: datacenterName,
		},
	)

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

func TestStatefulSet_ReconcileCapacity(t *testing.T) {
	cluster := getBaseInputCluster()
	cluster.Spec.Node.PersistentVolume = &v1alpha1.PersistentVolumeSpec{
		Capacity: corev1.ResourceList{
			"storage": kuberesource.MustParse("1000Gi"),
		},
	}

	expected := getBaseExpectedStatefulSet()
	expected.Spec.VolumeClaimTemplates[0].Spec.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: kuberesource.MustParse("1000Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceStorage: kuberesource.MustParse("1000Gi"),
		},
	}

	mockClient := &k8s.MockClient{}
	statefulset := getNewSS(cluster)
	got, err := statefulset.Reconcile(mockClient)

	assert.NoError(t, err)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
	}
}

// func TestStatefulSet_ExternalSeedsInitial(t *testing.T) {
// 	cluster := getBaseInputCluster()
// 	cluster.Spec.ExternalSeeds = []string{
// 		"external-seed-1.test.local",
// 		"external-seed-2.test.local",
// 	}

// 	expected := getBaseExpectedStatefulSet()
// 	expected.Spec.Template.Spec.Containers[0].Env[7].Value =
// 		"test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,external-seed-1.test.local,external-seed-2.test.local"

// 	mockClient := &k8s.MockClient{}
// 	statefulset := getNewSS(cluster)
// 	got, err := statefulset.Reconcile(mockClient)

// 	assert.NoError(t, err)
// 	if !reflect.DeepEqual(got, expected) {
// 		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
// 	}
// }

// func TestStatefulSet_ExternalSeedsSecondNode(t *testing.T) {
// 	cluster := getBaseInputCluster()
// 	cluster.Spec.ExternalSeeds = []string{
// 		"external-seed-1.test.local",
// 		"external-seed-2.test.local",
// 	}

// 	existing := getBaseExpectedStatefulSet()
// 	existing.Spec.Replicas = &one
// 	existing.ObjectMeta.ResourceVersion = "some-resource-version"
// 	existing.Status.ReadyReplicas = one

// 	expected := getBaseExpectedStatefulSet()
// 	expected.Spec.Template.Spec.Containers[0].Env[7].Value =
// 		"test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local,test-cluster-1-cassandra-1.some-service-name.test-namespace.svc.cluster.local,external-seed-1.test.local,external-seed-2.test.local"
// 	expected.ObjectMeta.ResourceVersion = "some-resource-version"
// 	expected.Spec.Replicas = &two
// 	expected.Spec.Template.Spec.Containers[0].Env[8].Value = "false"

// 	mockClient := &k8s.MockClient{
// 		GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
// 			return k8sutil.RuntimeObjectIntoRuntimeObject(existing, into)
// 		},
// 	}
// 	statefulset := getNewSS(cluster)
// 	got, err := statefulset.Reconcile(mockClient)

// 	assert.NoError(t, err)
// 	if !reflect.DeepEqual(got, expected) {
// 		t.Errorf("StatefulSet.Reconcile() = %v, want %v", got, expected)
// 	}
// }

func getNewSS(cluster *v1alpha1.CassandraCluster) *resource.StatefulSet {
	return resource.NewStatefulSet(
		cluster,
		resource.WithServiceAccountName("some-service-account-name"),
		resource.WithServiceName("some-service-name"),
	)
}

func getBaseInputCluster() *v1alpha1.CassandraCluster {
	return &v1alpha1.CassandraCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-1",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
		Spec: v1alpha1.ClusterSpec{
			Size: 2,
			Node: &v1alpha1.NodePolicy{
				Image:     "cassandra-image-test:1233",
				Resources: &corev1.ResourceRequirements{},
			},
		},
	}
}

func getBaseExpectedStatefulSet() *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster-1-cassandra",
			Namespace: "test-namespace",
			OwnerReferences: []metav1.OwnerReference{
				{Name: "test-cluster-1", Controller: &trueVar},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     "test-app",
					"cluster": "test-cluster-1",
					"state":   "serving",
					"type":    "cassandra-node",
				},
			},
			ServiceName: "some-service-name",
			Replicas:    &one,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.OnDeleteStatefulSetStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"type":    "cassandra-node",
						"state":   "serving",
						"cluster": "test-cluster-1",
						"app":     "test-app",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "some-service-account-name",
					Containers: []corev1.Container{
						{
							Name:            "cassandra",
							Image:           "cassandra-image-test:1233",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 7000, Name: "intra-node"},
								{ContainerPort: 7001, Name: "tls-intra-node"},
								{ContainerPort: 7199, Name: "jmx"},
								{ContainerPort: 9042, Name: "cql"},
								{ContainerPort: 9160, Name: "thrift"},
								{ContainerPort: 8778, Name: "metrics"},
							},
							Resources: corev1.ResourceRequirements{},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"IPC_LOCK"},
								},
							},
							Env: []corev1.EnvVar{
								{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
								{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}},
								{Name: "CASSANDRA_CLUSTER_NAME", Value: "test-cluster-1"},
								{Name: "SERVICE_NAME", Value: "some-service-name"},
								{Name: "CASSANDRA_ALLOCATE_TOKENS_FOR_KEYSPACE", Value: "test-cluster-1"},
								{Name: "CASSANDRA_MAX_HEAP", Value: "400M"},
								{Name: "CASSANDRA_MIN_HEAP", Value: "400M"},
								{Name: "CASSANDRA_SEEDS", Value: "test-cluster-1-cassandra-0.some-service-name.test-namespace.svc.cluster.local"},
								{Name: "CASSANDRA_AUTO_BOOTSTRAP", Value: "false"},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/bash",
											"-c",
											"/ready-probe.sh",
										},
									},
								},
								InitialDelaySeconds: fifteen,
								TimeoutSeconds:      five,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cassandra-keystore",
									MountPath: "/keystore",
								},
								{
									Name:      "test-cluster-1-cassandra-data",
									MountPath: "/var/lib/cassandra",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cassandra-keystore",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "test-cluster-1-cassandra-certs",
								},
							},
						},
						{
							Name: "jvm-agent-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "test-cluster-1-prometheus-jvm-agent-config",
									},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-1-cassandra-data",
						Annotations: map[string]string{
							"volume.beta.kubernetes.io/storage-class": "ssd",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: &ssd,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: capacity,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceStorage: capacity,
							},
						},
					},
				},
			},
		},
	}
}
