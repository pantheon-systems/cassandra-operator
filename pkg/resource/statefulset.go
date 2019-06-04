package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	readinessProbeInitialDelaySeconds = int32(15)
	readinessProbeTimeoutSeconds      = int32(5)
	defaultStorageCapacity            = resource.MustParse("1000Gi")
	readinessProbeScriptName          = "/ready-probe.sh"
	defaultStorageClassName           = ssdStorageClassName
)

// StatefulSet is a reconciller for apps/v1 StatefulSet
type StatefulSet struct {
	desired *appsv1.StatefulSet
	cluster *v1alpha1.CassandraCluster

	seedList            []string
	desiredReplicas     int32
	enableAutoBootstrap bool

	options *builderOp
}

// NewStatefulSet constructs a new StatefulSet Reconciler
func NewStatefulSet(cc *v1alpha1.CassandraCluster, opts ...BuilderOption) *StatefulSet {
	op := newBuilderOp()
	op.applyOpts(opts)

	return &StatefulSet{
		cluster:             cc,
		options:             op,
		enableAutoBootstrap: true,
		seedList:            []string{},
	}
}

// Reconcile merges the desired state with the actual state
func (b *StatefulSet) Reconcile(ctx context.Context, driver client.Client) (runtime.Object, error) {
	var err error

	if b.options.ServiceAccountName == "" || b.options.ServiceName == "" {
		return nil, fmt.Errorf("both ServiceAccountName and ServiceAccount are required: %s, %s", b.options.ServiceAccountName, b.options.ServiceName)
	}

	objectMeta := b.buildObjectMeta()

	namespacedName := types.NamespacedName{
		Name:      objectMeta.GetName(),
		Namespace: objectMeta.GetNamespace(),
	}

	existing := &appsv1.StatefulSet{}
	err = driver.Get(ctx, namespacedName, existing)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, errors.New("could not get existing")
	}

	if existing.ResourceVersion == "" {
		b.desiredReplicas = int32(1)
		b.calculateSeedList(b.desiredReplicas)
		// we use 0s here cuase we have no existing replicas and no ready replicas
		b.calculateAutoBootstrap(0, 0)
		b.configureDesired()

		err = driver.Create(ctx, b.desired)

		return b.desired, err
	}

	existingReplicas := *existing.Spec.Replicas
	existingReadyReplicas := existing.Status.ReadyReplicas

	// TODO: can capture the second return val for this method which is a bool to repair or not
	b.desiredReplicas, _ = b.calculateReplicas(existingReplicas, existingReadyReplicas)
	b.calculateAutoBootstrap(existingReplicas, existingReadyReplicas)
	b.calculateSeedList(b.desiredReplicas)

	b.configureDesired()

	b.desired.ResourceVersion = existing.ResourceVersion
	// We are using Update here as we have the OnDelete update stratagy in place for the stateful set
	// See https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/#updating-statefulsets
	err = driver.Update(ctx, b.desired)

	if err != nil {
		return nil, err
	}

	return b.desired, nil
}

// Calculates seed list for cluster. If ExternalSeeds is set in the resource
// We append the external seeds to the primary cluster seed list.
// TODO: Do not make all local cluster nodes seeds
func (b *StatefulSet) calculateSeedList(replicas int32) {
	cassandraSeedsList := []string{}
	for i := int32(0); i < replicas; i++ {
		// ex. test-cluster-cassandra-1.test-cluster.sandbox-foo.svc.cluster.local
		seed := fmt.Sprintf("%s-cassandra-%d.%s.%s.svc.cluster.local",
			b.cluster.GetName(),
			int(i),
			b.options.ServiceName,
			b.cluster.GetNamespace())
		cassandraSeedsList = append(cassandraSeedsList, seed)
	}

	if b.cluster.Spec.ExternalSeeds != nil || len(b.cluster.Spec.ExternalSeeds) > 0 {
		cassandraSeedsList = append(cassandraSeedsList, b.cluster.Spec.ExternalSeeds...)
	}

	b.seedList = cassandraSeedsList
}

// Calculates auto_bootstrap value for cluster
// auto_bootstrap:
// (Default: true) This setting has been removed from default configuration. It makes new
// (non-seed) nodes automatically migrate the right data to themselves. When initializing
// a fresh cluster without data, add auto_bootstrap: false.
func (b *StatefulSet) calculateAutoBootstrap(existingReplicas, existingReadyReplicas int32) {
	// if we specify the external seeds we are creating a new DC for an existing
	// topology
	isMultiDC := len(b.cluster.Spec.ExternalSeeds) > 0

	// we are creating a new ring (DC/cluster) of an existing cassandra setup and multi-dc is true
	// or we are initilizing a new cluster with no data and this is the first node
	if existingReplicas == 0 || b.desiredReplicas == 1 || isMultiDC {
		b.enableAutoBootstrap = false
	}

	// if we have 1 replica ready but we are supposed to have more than one
	// and we have auto_bootstrap set to false, we should change it to true
	// only if we are not creating a new data center (ring) as specified by having
	// ExternalSeeds set in the CRD
	if existingReplicas == 1 && existingReadyReplicas == 1 &&
		b.cluster.Spec.Size > 1 && !isMultiDC {
		b.enableAutoBootstrap = true
	}
}

/*
	Possible States:
		existing ready | existing expected | statefulset expected | action
			0					1					anything		first node startup mode, auto-bootstrap is false
			1					1					  > 1			intial scaling up or one node cluster scaling up, have a ready first node, auto-bootstrap is true
			x					x					   x			all nodes are create and ready, no action, auto-bootstrap is true
			x					x					   y			scaling up or down from any exisiting ready state, if the |y-x| > 1 then scale one pod at a time
			x					y					   y			in process of scaling up, waiting for new pod to be ready
*/
func (b *StatefulSet) calculateReplicas(existingReplicas, existingReadyReplicas int32) (int32, bool) {
	expectedReplicas := int32(b.cluster.Spec.Size)
	replicaDelta := expectedReplicas - existingReplicas

	// First node is creating (0 ready), but we are expecting more, first node has to auto-bootstrap (set in create code)
	if existingReadyReplicas == 0 && existingReplicas != 0 {
		// still waiting for first node to come online, we got here because an event triggered a change
		return int32(1), false
	}

	replicas := int32(b.cluster.Spec.Size)
	repair := false
	// all existing are ready, we are either removing or adding a node
	if existingReadyReplicas == existingReplicas &&
		existingReplicas != expectedReplicas {
		// the difference in the number of nodes is greater than one, we need to only scale up or down
		// one at a time
		replicas = expectedReplicas
		if replicaDelta > 1 {
			replicas = existingReplicas + 1
		} else if replicaDelta < 1 {
			replicas = existingReplicas - 1
		}
		repair = true
	}

	return replicas, repair
}

func (b *StatefulSet) buildObjectMeta() metav1.ObjectMeta {
	statefulSetName := fmt.Sprintf("%s-cassandra", b.cluster.GetName())
	return metav1.ObjectMeta{
		Name:      statefulSetName,
		Namespace: b.cluster.GetNamespace(),
	}
}

// buildDesired creates and configures the desired statefulset
func (b *StatefulSet) configureDesired() {
	b.desired = &appsv1.StatefulSet{
		TypeMeta:   GetStatefulSetTypeMeta(),
		ObjectMeta: b.buildObjectMeta(),
		Spec: appsv1.StatefulSetSpec{
			// See https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/#updating-statefulsets
			// We are choosing to manage the restart of the pods ourselves, this is due to the nature of cassandra
			// when a new node is started it streams the data from the other node, if we do a rolling restart at that point
			// it will interupt the streaming and cassandra can have streaming failures to the new nodes aborting the streaming
			// process
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.OnDeleteStatefulSetStrategyType,
			},
			Selector:    &metav1.LabelSelector{},
			ServiceName: b.options.ServiceName,
			Replicas:    &b.desiredReplicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
			},
		},
	}

	b.buildDesiredCassandraPodSpec()
	b.buildLabels()
	controllerutil.SetControllerReference(b.cluster, b.desired, scheme.Scheme)
	b.buildVolumeClaimTemplates()

	// TODO: JvmAgent should be itoa or string enum
	if b.cluster.Spec.JvmAgent == "sidecar" {
		if b.desired.Spec.Template.ObjectMeta.Annotations == nil {
			b.desired.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}
		b.desired.Spec.Template.ObjectMeta.Annotations["prometheus.io/scrape"] = "true"
		b.desired.Spec.Template.ObjectMeta.Annotations["prometheus.io/port"] = "9126"
	}
}

func (b *StatefulSet) buildLabels() {
	labels := map[string]string{
		"cluster": b.cluster.GetName(),
		"type":    "cassandra-node",
		"state":   "serving",
	}

	if appName, ok := b.cluster.GetLabels()["app"]; ok {
		labels["app"] = appName
	}

	b.desired.Spec.Selector.MatchLabels = labels
	b.desired.Spec.Template.ObjectMeta.SetLabels(labels)
}

func (b *StatefulSet) buildVolumeClaimTemplates() {
	pvSpec := b.cluster.Spec.Node.PersistentVolume

	storageClassName := defaultStorageClassName
	if pvSpec != nil && pvSpec.StorageClassName != "" {
		storageClassName = pvSpec.StorageClassName
	}

	capacity := defaultStorageCapacity
	if pvSpec != nil && pvSpec.Capacity != nil {
		if storage, ok := pvSpec.Capacity["storage"]; ok {
			capacity = storage
		}
	}

	b.desired.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-cassandra-data", b.cluster.GetName()),
				Annotations: map[string]string{
					"volume.beta.kubernetes.io/storage-class": storageClassName,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: capacity,
					},
					Limits: corev1.ResourceList{
						corev1.ResourceStorage: capacity,
					},
				},
			},
		},
	}
}
