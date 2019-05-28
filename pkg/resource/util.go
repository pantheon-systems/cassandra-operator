package resource

import (
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mergeMap(a, b map[string]string) map[string]string {
	for key, value := range b {
		a[key] = value
	}
	return a
}

func asOwner(m *v1alpha1.CassandraCluster) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}
