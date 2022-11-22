package resource_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/resource"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
)

var (
	twoObj = intstr.Parse("2")
)

func TestPodDisruptionBudget_Reconcile(t *testing.T) {
	type fields struct {
		cluster *v1alpha1.CassandraCluster
		actual  *policyv1beta1.PodDisruptionBudget
	}
	tests := []struct {
		name    string
		fields  fields
		want    runtime.Object
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
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
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
							{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
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
						{Name: "test-cluster-1", APIVersion: "v1", Kind: "CassandraCluster"},
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
			b := resource.NewPodDisruptionBudget(tt.fields.cluster)
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
