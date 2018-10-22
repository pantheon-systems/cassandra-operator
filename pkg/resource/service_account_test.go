package resource_test

import (
	"errors"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccount_Reconcile(t *testing.T) {
	type fields struct {
		actual          *corev1.ServiceAccount
		cluster         *v1alpha1.CassandraCluster
		mockGetError    error
		mockCreateError error
	}
	tests := []struct {
		name    string
		fields  fields
		want    sdk.Object
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
						{Name: "test-cluster-1", Controller: &trueVar},
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
				mockCreateError: k8serrors.NewAlreadyExists(
					schema.GroupResource{Group: "", Resource: "serviceaccounts"},
					"test-cluster-1-service-account",
				),
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
						{Name: "test-cluster-1", Controller: &trueVar},
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
			name: "get-error",
			fields: fields{
				actual:       nil,
				cluster:      &v1alpha1.CassandraCluster{},
				mockGetError: errors.New("some other error"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "create-error",
			fields: fields{
				actual:          nil,
				cluster:         &v1alpha1.CassandraCluster{},
				mockCreateError: errors.New("some other error"),
			},
			want:    nil,
			wantErr: true,
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
					if tt.fields.mockCreateError != nil {
						return tt.fields.mockCreateError
					}
					return nil
				},
			}

			b := resource.NewServiceAccount(tt.fields.cluster)
			got, err := b.Reconcile(mockClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("ServiceAccount.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ServiceAccount.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
