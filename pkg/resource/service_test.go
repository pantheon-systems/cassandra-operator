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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ServiceUnsupportedType resource.ClusterServiceType = 40000
)

func TestService_Reconcile(t *testing.T) {
	type fields struct {
		mockGetError            error
		mockCreateOrUpdateError error
		actual                  *corev1.Service
		cluster                 *v1alpha1.CassandraCluster
		options                 []resource.BuilderOption
	}
	tests := []struct {
		name    string
		fields  fields
		want    sdk.Object
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
			name: "get error",
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
				mockGetError: errors.New("some error"),
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
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
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
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
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
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
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
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
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
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
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
			mockClient := &k8s.MockClient{
				GetCallback: func(into sdk.Object, opts ...sdk.GetOption) error {
					if tt.fields.mockGetError != nil {
						return tt.fields.mockGetError
					}

					if tt.fields.actual != nil {
						if err := k8sutil.RuntimeObjectIntoRuntimeObject(tt.fields.actual, into); err != nil {
							return err
						}
					}

					return nil
				},
				CreateCallback: func(object sdk.Object) error {
					if tt.fields.mockCreateOrUpdateError != nil {
						return tt.fields.mockCreateOrUpdateError
					}
					return nil
				},
				UpdateCallback: func(object sdk.Object) error {
					if tt.fields.mockCreateOrUpdateError != nil {
						return tt.fields.mockCreateOrUpdateError
					}
					return nil
				},
			}

			b := resource.NewService(tt.fields.cluster, tt.fields.options...)
			got, err := b.Reconcile(mockClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
