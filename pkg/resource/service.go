package resource

import (
	"errors"
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	opsdk "github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterServiceType represents the different services that this
// builder group can create
type ClusterServiceType int

const (
	// ServiceTypeNone represents the default value and does not correlate to a service type
	ServiceTypeNone ClusterServiceType = iota
	// ServiceTypePublicLB represents the public load ballanced service
	ServiceTypePublicLB
	// ServiceTypePublicPod represents the public service that points at a single pod
	ServiceTypePublicPod
	// ServiceTypeHeadless represents the headless services for the nodes
	ServiceTypeHeadless
	// ServiceTypeInternal represents the load ballanced service that is only accessable in the cluster
	ServiceTypeInternal
)

// Service is a reconciller for a k8s core/v1 service resource
type Service struct {
	configured *corev1.Service
	cluster    *v1alpha1.CassandraCluster
	options    *builderOp
}

// NewService is the constructor for ServiceBuilder class
func NewService(cc *v1alpha1.CassandraCluster, opts ...BuilderOption) *Service {
	op := newBuilderOp()
	op.applyOpts(opts)

	return &Service{
		cluster: cc,
		options: op,
	}
}

// Reconcile takes the current state and creates an object that will reconcile that
// to the desired state
func (b *Service) Reconcile(driver opsdk.Client) (sdk.Object, error) {
	var err error

	if b.options.ServiceType == ServiceTypeNone {
		return nil, fmt.Errorf("invalid service type: %d", b.options.ServiceType)
	}

	err = b.buildConfigured(b.options.ServiceType)
	if err != nil {
		return nil, err
	}

	existing := &corev1.Service{
		TypeMeta:   GetServiceTypeMeta(),
		ObjectMeta: b.configured.ObjectMeta,
	}
	err = driver.Get(existing)
	if err != nil {
		return nil, errors.New("could not get existing")
	}

	if existing.ResourceVersion != "" {
		// here we have one that is existing and one that is expected
		// we put our code here to reconcile the two and return
		// the reconciled object
		b.configured.ResourceVersion = existing.ResourceVersion
		b.configured.Spec.ClusterIP = existing.Spec.ClusterIP
		err = driver.Update(b.configured)
	} else {
		err = driver.Create(b.configured)
	}

	if err != nil {
		return nil, err
	}

	return b.configured, nil
}

// Build constructs and persists the sdk object to kube
func (b *Service) buildConfigured(clusterServiceType ClusterServiceType) error {
	b.configured = &corev1.Service{
		TypeMeta: GetServiceTypeMeta(),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{},
		},
	}

	switch clusterServiceType {
	case ServiceTypePublicLB:
		b.configurePublicLB()
	case ServiceTypePublicPod:
		b.configurePublicPod()
	case ServiceTypeHeadless:
		b.configureHeadless()
	case ServiceTypeInternal:
		b.configureInternal()
	default:
		return errors.New("Unsupported Service type")
	}

	b.configureDefaultSelectors()
	b.configureDefaultLabels()

	b.setOwner(asOwner(b.cluster))
	return nil
}

func (b *Service) configurePublicLB() {
	b.configured.ObjectMeta = metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-cassandra-public", b.cluster.GetName()),
		Namespace: b.cluster.GetNamespace(),
		Labels:    map[string]string{},
	}

	b.configured.Spec.Type = corev1.ServiceTypeLoadBalancer
	b.configured.Spec.Ports = []corev1.ServicePort{
		{
			Port: 9042,
			Name: "cql",
		},
		{
			Port: 9160,
			Name: "thrift",
		},
	}

	labels := b.configured.GetLabels()
	labels["service-type"] = "public"
	b.configured.SetLabels(labels)
}

func (b *Service) configurePublicPod() {
	clusterName := b.cluster.GetName()
	podNumber := b.options.PodNumber

	b.configured.ObjectMeta = metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-cassandra-public-%d", clusterName, podNumber),
		Namespace: b.cluster.GetNamespace(),
		Labels:    map[string]string{},
	}
	b.configured.Spec.Type = corev1.ServiceTypeLoadBalancer
	b.configured.Spec.Ports = []corev1.ServicePort{
		{
			Port: 7001,
			Name: "ssl-internode-cluster",
		},
	}

	b.configured.Spec.Selector["statefulset.kubernetes.io/pod-name"] = fmt.Sprintf("%s-cassandra-%d", clusterName, podNumber)

	labels := b.configured.GetLabels()
	labels["service-type"] = "public-pod"
	b.configured.SetLabels(labels)
}

func (b *Service) configureHeadless() {
	b.configured.ObjectMeta = metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-cassandra-headless", b.cluster.GetName()),
		Namespace: b.cluster.GetNamespace(),
		Labels:    map[string]string{},
	}

	b.configured.Spec.ClusterIP = corev1.ClusterIPNone
	b.configured.Spec.Ports = []corev1.ServicePort{
		{
			Port: 9042,
			Name: "cql",
		},
		{
			Port: 9160,
			Name: "thrift",
		},
	}

	labels := b.configured.GetLabels()
	labels["service-type"] = "headless"
	b.configured.SetLabels(labels)
}

func (b *Service) configureInternal() {
	b.configured.ObjectMeta = metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-cassandra", b.cluster.GetName()),
		Namespace: b.cluster.GetNamespace(),
		Labels:    map[string]string{},
	}

	b.configured.Spec.Type = corev1.ServiceTypeClusterIP
	b.configured.Spec.Ports = []corev1.ServicePort{
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
	}

	labels := b.configured.GetLabels()
	labels["service-type"] = "internal"
	b.configured.SetLabels(labels)
}

func (b *Service) configureDefaultLabels() {
	serviceLabels := b.configured.GetLabels()
	clusterLabels := b.cluster.GetLabels()

	serviceLabels["cluster"] = b.cluster.GetName()

	b.configured.SetLabels(mergeMap(serviceLabels, clusterLabels))
}

func (b *Service) configureDefaultSelectors() {
	b.configured.Spec.Selector["cluster"] = b.cluster.GetName()
	b.configured.Spec.Selector["state"] = "serving"

	if appName, ok := b.cluster.GetLabels()["app"]; ok {
		b.configured.Spec.Selector["app"] = appName
	}
}

func (b *Service) setOwner(owner metav1.OwnerReference) {
	b.configured.SetOwnerReferences(append(b.configured.GetOwnerReferences(), owner))
}
