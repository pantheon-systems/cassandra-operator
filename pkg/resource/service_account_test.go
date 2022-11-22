package resource_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestServiceAccount_Reconcile(t *testing.T) {
	type fields struct {
		actual  *corev1.ServiceAccount
		cluster *v1alpha1.CassandraCluster
	}
	tests := []struct {
		name    string
		fields  fields
		want    runtime.Object
		wantErr bool
	}{
		{
			name: "does-not-exist",
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
			},
			want: &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-service-account",
					Namespace: "test-namespace",
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "quayio",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "already-exists",
			fields: fields{
				actual: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cluster-1",
						Namespace:       "test-namespace",
						ResourceVersion: "test-resource-version",
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
			},
			want: &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-cluster-1-service-account",
					Namespace:       "test-namespace",
					ResourceVersion: "test-resource-version",
					OwnerReferences: []metav1.OwnerReference{
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
					},
				},
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "quayio",
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
			b := resource.NewServiceAccount(tt.fields.cluster)
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
				t.Errorf("ServiceAccount.Reconcile() = %v, want %v", result, tt.want)
			}
		})
	}
}
