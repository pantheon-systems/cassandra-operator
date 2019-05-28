package resource

const (
	cronJobAPIVersion             = "batch/v1beta1"
	cronJobKind                   = "CronJob"
	cronJobNameTemplate           = "%s-cassandra-repair"
	serviceAPIVersion             = "v1"
	serviceKind                   = "Service"
	statefulSetAPIVersion         = "apps/v1"
	statefulSetKind               = "StatefulSet"
	serviceAccountAPIVersion      = "v1"
	serviceAccountKind            = "ServiceAccount"
	podAPIVersion                 = "v1"
	podKind                       = "Pod"
	podDisruptionBudgetAPIVersion = "policy/v1beta1"
	podDisruptionBudgetKind       = "PodDisruptionBudget"
	cassandraClusterAPIVersion    = "database.pantheon.io/v1alpha1"
	cassandraClusterKind          = "CassandraCluster"

	kubeNamespaceEnvVar    = "KUBE_NAMESPACE"
	cassandraClusterEnvVar = "CASSANDRA_CLUSTER"
	appNameEnvVar          = "APP_NAME"

	ssdStorageClassName = "ssd"
)
