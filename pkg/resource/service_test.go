package resource_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	ServiceUnsupportedType resource.ClusterServiceType = 40000
)

func TestService_Reconcile(t *testing.T) {
	type fields struct {
		actual  *corev1.Service
		cluster *v1alpha1.CassandraCluster
		options []resource.BuilderOption
	}
	tests := []struct {
		name    string
		fields  fields
		want    runtime.Object
		wantErr bool
	}{
		{
			name: "unsupported service type",
			fields: fields{
				actual: nil,
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{
					resource.WithServiceType(ServiceUnsupportedType),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "serviceType not set",
			fields: fields{
				actual: nil,
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "any-kind-already-exists",
			fields: fields{
				actual: &corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cluster-1-cassandra-public",
						Namespace:       "test-namespace",
						ResourceVersion: "existing-resource-version",
					},
				},
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{
					resource.WithServiceType(resource.ServiceTypePublicLB),
				},
			},
			want: &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra-public",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster":      "test-cluster-1",
						"app":          "test-app",
						"service-type": "public",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
					ResourceVersion: "existing-resource-version",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{
						"cluster": "test-cluster-1",
						"state":   "serving",
						"app":     "test-app",
					},
					Ports: []corev1.ServicePort{
						{
							Port: 9042,
							Name: "cql",
						},
						{
							Port: 9160,
							Name: "thrift",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "public-lb-does-not-exist-create",
			fields: fields{
				actual: nil,
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{
					resource.WithServiceType(resource.ServiceTypePublicLB),
				},
			},
			want: &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra-public",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster":      "test-cluster-1",
						"app":          "test-app",
						"service-type": "public",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{
						"cluster": "test-cluster-1",
						"state":   "serving",
						"app":     "test-app",
					},
					Ports: []corev1.ServicePort{
						{
							Port: 9042,
							Name: "cql",
						},
						{
							Port: 9160,
							Name: "thrift",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "public-pod-does-not-exist-create",
			fields: fields{
				actual: nil,
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{
					resource.WithServiceType(resource.ServiceTypePublicPod),
					resource.WithPodNumber(1),
				},
			},
			want: &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra-public-1",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster":      "test-cluster-1",
						"app":          "test-app",
						"service-type": "public-pod",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
					Selector: map[string]string{
						"cluster":                            "test-cluster-1",
						"state":                              "serving",
						"app":                                "test-app",
						"statefulset.kubernetes.io/pod-name": "test-cluster-1-cassandra-1",
					},
					Ports: []corev1.ServicePort{
						{
							Port: 7001,
							Name: "ssl-internode-cluster",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "headless-does-not-exist-create",
			fields: fields{
				actual: nil,
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{
					resource.WithServiceType(resource.ServiceTypeHeadless),
				},
			},
			want: &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra-headless",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster":      "test-cluster-1",
						"app":          "test-app",
						"service-type": "headless",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: corev1.ClusterIPNone,
					Selector: map[string]string{
						"cluster": "test-cluster-1",
						"state":   "serving",
						"app":     "test-app",
					},
					Ports: []corev1.ServicePort{
						{
							Port: 9042,
							Name: "cql",
						},
						{
							Port: 9160,
							Name: "thrift",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "internal-does-not-exist-create",
			fields: fields{
				actual: nil,
				cluster: &v1alpha1.CassandraCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-1",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"app": "test-app",
						},
					},
				},
				options: []resource.BuilderOption{
					resource.WithServiceType(resource.ServiceTypeInternal),
				},
			},
			want: &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"cluster":      "test-cluster-1",
						"app":          "test-app",
						"service-type": "internal",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Selector: map[string]string{
						"cluster": "test-cluster-1",
						"state":   "serving",
						"app":     "test-app",
					},
					Ports: []corev1.ServicePort{
						{
							Port: 9042,
							Name: "cql",
						},
						{
							Port: 9160,
							Name: "thrift",
						},
						{
							Port: 8778,
							Name: "metrics",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scheme.Scheme
			s.AddKnownTypes(corev1.SchemeGroupVersion, tt.fields.cluster)

			objs := []runtime.Object{}
			if tt.fields.actual != nil {
				objs = append(objs, tt.fields.actual)
			}

			mockClient := fake.NewFakeClient(objs...)
			b := resource.NewService(tt.fields.cluster, tt.fields.options...)
			result, err := b.Reconcile(context.TODO(), mockClient)

			if tt.wantErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}

			// we need to ignore
			//   ObjectMeta.OwnerReferences.Controller
			//   ObjectMeta.OwnerReferences.BlockOwnerDeletion
			ctrlOwnerRef := reflect.ValueOf(result).Elem().FieldByName("ObjectMeta").FieldByName("OwnerReferences").Index(0)
			ctrl := ctrlOwnerRef.FieldByName("Controller")
			ctrl.Set(reflect.Zero(ctrl.Type()))
			blockOwnerDeletion := ctrlOwnerRef.FieldByName("BlockOwnerDeletion")
			blockOwnerDeletion.Set(reflect.Zero(blockOwnerDeletion.Type()))

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("Service.Reconcile() = %v, want %v", result, tt.want)
			}
		})
	}
}
