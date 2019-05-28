package resource

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"strings"
)

func (b *StatefulSet) buildDesiredCassandraPodSpec() {
	b.desired.Spec.Template.Spec = corev1.PodSpec{
		ServiceAccountName: b.options.ServiceAccountName,
		Containers:         []corev1.Container{},
	}

	b.buildPodVolumes()
	b.buildCassandraContainer()

	if b.cluster.Spec.JvmAgent == "sidecar" {
		b.buildTelegrafContainer()
	}

	if b.cluster.Spec.Affinity != nil {
		b.desired.Spec.Template.Spec.Affinity = b.cluster.Spec.Affinity
	}
}

func (b *StatefulSet) buildPodVolumes() {
	secretName := fmt.Sprintf("%s-cassandra-certs", b.cluster.GetName())
	if b.cluster.Spec.SecretName != "" {
		secretName = b.cluster.Spec.SecretName
	}

	jvmAgentConfigName := fmt.Sprintf("%s-prometheus-jvm-agent-config", b.cluster.GetName())
	if b.cluster.Spec.JvmAgentConfigName != "" {
		jvmAgentConfigName = b.cluster.Spec.JvmAgentConfigName
	}

	b.desired.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "cassandra-keystore",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		},
		{
			Name: "jvm-agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: jvmAgentConfigName,
					},
				},
			},
		},
	}
}

func (b *StatefulSet) buildTelegrafContainer() {
	// telegraf metrics sidecar
	// https://hub.docker.com/_/telegraf/
	container := corev1.Container{
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
			// mount cassandra's persistent-disk into the telegraf pod so that telegraf can collect usage metrics
			{
				Name:      fmt.Sprintf("%s-cassandra-data", b.cluster.GetName()),
				MountPath: b.getFileMountPath(),
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("0.1"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
	}
	b.desired.Spec.Template.Spec.Containers = append(b.desired.Spec.Template.Spec.Containers, container)
}

func (b *StatefulSet) buildCassandraContainer() {
	container := corev1.Container{
		Name:            "cassandra",
		Image:           b.cluster.Spec.Node.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports:           b.buildContainerPorts(),
		Resources:       *b.cluster.Spec.Node.Resources,
		SecurityContext: b.buildSecurityContext(),
		Env:             b.buildEnvVars(),
		ReadinessProbe:  b.buildReadinessProbe(),
		VolumeMounts:    b.buildContainerVolumeMounts(),
	}
	b.desired.Spec.Template.Spec.Containers = append(b.desired.Spec.Template.Spec.Containers, container)
}

func (b *StatefulSet) buildSecurityContext() *corev1.SecurityContext {
	// allows the cassandra pod to run with JNA and use mlockall
	// cassandra really wants to manage its own memory ;)
	// http://docs.datastax.com/en/archived/cassandra/1.2/cassandra/install/installJnaDeb.html
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{
				"IPC_LOCK",
			},
		},
	}
}

func (b *StatefulSet) getFileMountPath() string {
	fileMountPath := "/var/lib/cassandra"
	if b.cluster.Spec.Node != nil && b.cluster.Spec.Node.FileMountPath != "" {
		fileMountPath = b.cluster.Spec.Node.FileMountPath
	}
	return fileMountPath
}

func (b *StatefulSet) buildContainerVolumeMounts() []corev1.VolumeMount {
	fileMountPath := b.getFileMountPath()

	mounts := []corev1.VolumeMount{
		{
			Name:      "cassandra-keystore",
			MountPath: "/keystore",
		},
		{
			Name:      fmt.Sprintf("%s-cassandra-data", b.cluster.GetName()),
			MountPath: fileMountPath,
		},
	}

	// if we do jvm agent then we need to load the prom jvm agent config
	// into the cassandra container
	if b.cluster.Spec.JvmAgent == "jvm" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "jvm-agent-config",
			MountPath: "/jvm-agent",
		})
	}

	return mounts
}

func (b *StatefulSet) buildContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			ContainerPort: 7000,
			Name:          "intra-node",
		},
		{
			ContainerPort: 7001,
			Name:          "tls-intra-node",
		},
		{
			ContainerPort: 7199,
			Name:          "jmx",
		},
		{
			ContainerPort: 9042,
			Name:          "cql",
		},
		{
			ContainerPort: 9160,
			Name:          "thrift",
		},
		{
			ContainerPort: 8778,
			Name:          "metrics",
		},
	}
}

func (b *StatefulSet) buildReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/bash",
					"-c",
					readinessProbeScriptName,
				},
			},
		},
		InitialDelaySeconds: readinessProbeInitialDelaySeconds,
		TimeoutSeconds:      readinessProbeTimeoutSeconds,
	}
}

func (b *StatefulSet) buildEnvVars() []corev1.EnvVar {
	// keyspace name for the initial keyspace, if not set use cluster name
	keyspaceName := b.cluster.Spec.KeyspaceName
	if keyspaceName == "" {
		keyspaceName = b.cluster.GetName()
	}

	vars := []corev1.EnvVar{
		// we need to namespace to work around the jvm resolver not honoring search domains in the contaienr.
		// the run.sh will fully qualify hte discovery name for the first host based on clsuter name namespace and serviceName
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		// for many of the cassandra listeners they want to bind to an ip, so we have to pass it down.
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name:  "CASSANDRA_CLUSTER_NAME",
			Value: b.cluster.GetName(),
		},
		{
			Name:  "SERVICE_NAME",
			Value: b.options.ServiceName,
		},
		{
			Name:  "CASSANDRA_ALLOCATE_TOKENS_FOR_KEYSPACE",
			Value: keyspaceName,
		},
		{
			Name:  "CASSANDRA_MAX_HEAP",
			Value: "400M",
		},
		{
			Name:  "CASSANDRA_MIN_HEAP",
			Value: "400M",
		},
		{
			Name:  "CASSANDRA_SEEDS",
			Value: strings.Join(b.seedList, ","),
		},
		{
			Name:  "CASSANDRA_AUTO_BOOTSTRAP",
			Value: strconv.FormatBool(b.enableAutoBootstrap),
		},
	}

	if b.cluster.Spec.Datacenter != "" {
		vars = append(vars,
			corev1.EnvVar{
				Name:  "CASSANDRA_DC",
				Value: b.cluster.Spec.Datacenter,
			})
	}

	return vars
}
