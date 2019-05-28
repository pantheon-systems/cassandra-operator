package resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetServiceTypeMeta returns meta/v1 TypeMeta for core/v1 Service
func GetServiceTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: serviceAPIVersion,
		Kind:       serviceKind,
	}
}

// GetServiceAccountTypeMeta returns meta/v1 TypeMeta for core/v1 ServiceAccount
func GetServiceAccountTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: serviceAccountAPIVersion,
		Kind:       serviceAccountKind,
	}
}

// GetStatefulSetTypeMeta returns meta/v1 TypeMeta for apps/v1 StatefulSet
func GetStatefulSetTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: statefulSetAPIVersion,
		Kind:       statefulSetKind,
	}
}

// GetCronJobTypeMeta returns meta/v1 TypeMeta for batch/v1beta1 CronJob
func GetCronJobTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: cronJobAPIVersion,
		Kind:       cronJobKind,
	}
}

// GetPodTypeMeta returns meta/v1 TypeMeta for core/v1 Pod
func GetPodTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: podAPIVersion,
		Kind:       podKind,
	}
}

// GetPodDisruptionBudgetTypeMeta returns meta/v1 TypeMeta for policy/v1beta1 PodDisruptionBudget
func GetPodDisruptionBudgetTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: podDisruptionBudgetAPIVersion,
		Kind:       podDisruptionBudgetKind,
	}
}

// GetCassandraClusterTypeMeta returns meta/v1 TypeMeta for v1alpha1 CassandraCluster
func GetCassandraClusterTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: cassandraClusterAPIVersion,
		Kind:       cassandraClusterKind,
	}
}
