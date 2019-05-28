package resource_test

import (
	"errors"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"testing"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
)

var (
	twoObj = intstr.Parse("2")
)

func TestPodDisruptionBudget_Reconcile(t *testing.T) {
	type fields struct {
		cluster         *v1alpha1.CassandraCluster
		actual          *policyv1beta1.PodDisruptionBudget
		mockGetError    error
		mockUpdateError error
		mockCreateError error
	}
	tests := []struct {
		name    string
		fields  fields
		want    sdk.Object
		wantErr bool
	}{
		{
			name: "new",
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
						EnablePodDisruptionBudget: true,
					},
				},
				actual: nil,
			},
			want: &policyv1beta1.PodDisruptionBudget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy/v1beta1",
					Kind:       "PodDisruptionBudget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster-1-cassandra",
					Namespace: "test-namespace",
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
					},
				},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &twoObj,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"cluster": "test-cluster-1",
							"state":   "serving",
							"app":     "test-app",
						},
					},
				},
			},
		},
		{
			name: "update",
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
						EnablePodDisruptionBudget: true,
					},
				},
				actual: &policyv1beta1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "policy/v1beta1",
						Kind:       "PodDisruptionBudget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cluster-1-cassandra",
						Namespace:       "test-namespace",
						ResourceVersion: "test-resource-version",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-cluster-1",
								Controller: &trueVar,
							},
						},
					},
					Spec: policyv1beta1.PodDisruptionBudgetSpec{
						MinAvailable: &twoObj,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"cluster": "test-cluster-1",
								"state":   "serving",
								"app":     "test-app",
							},
						},
					},
				},
			},
			want: &policyv1beta1.PodDisruptionBudget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy/v1beta1",
					Kind:       "PodDisruptionBudget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-cluster-1-cassandra",
					Namespace:       "test-namespace",
					ResourceVersion: "test-resource-version",
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "test-cluster-1",
							Controller: &trueVar,
						},
					},
				},
				Spec: policyv1beta1.PodDisruptionBudgetSpec{
					MinAvailable: &twoObj,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"cluster": "test-cluster-1",
							"state":   "serving",
							"app":     "test-app",
						},
					},
				},
			},
		},
		{
			name: "get-error",
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
						EnablePodDisruptionBudget: true,
					},
				},
				mockGetError: errors.New("some error"),
				actual:       nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "create-error",
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
						EnablePodDisruptionBudget: true,
					},
				},
				actual:          nil,
				mockCreateError: errors.New("error creating"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "update-error",
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
						EnablePodDisruptionBudget: true,
					},
				},
				actual: &policyv1beta1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "policy/v1beta1",
						Kind:       "PodDisruptionBudget",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cluster-1-cassandra",
						Namespace:       "test-namespace",
						ResourceVersion: "test-resource-version",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-cluster-1",
								Controller: &trueVar,
							},
						},
					},
					Spec: policyv1beta1.PodDisruptionBudgetSpec{
						MinAvailable: &twoObj,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"cluster": "test-cluster-1",
								"state":   "serving",
								"app":     "test-app",
							},
						},
					},
				},
				mockUpdateError: errors.New("update error"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKubeClient := &k8s.MockClient{
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
				UpdateCallback: func(object sdk.Object) error {
					if tt.fields.mockUpdateError != nil {
						return tt.fields.mockUpdateError
					}
					return nil
				},
			}
			b := resource.NewPodDisruptionBudget(tt.fields.cluster)
			got, err := b.Reconcile(mockKubeClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("PodDisruptionBudget.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodDisruptionBudget.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
