package resource

import (
	"errors"
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	opsdk "github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccount is a reconciller for configured vs actual for service account
type ServiceAccount struct {
	configured *corev1.ServiceAccount
	cluster    *v1alpha1.CassandraCluster
}

// NewServiceAccount creates a new ServiceAccount
func NewServiceAccount(cc *v1alpha1.CassandraCluster) *ServiceAccount {
	return &ServiceAccount{
		cluster: cc,
	}
}

// Reconcile merges the actual state with the desired state to reconcile the core/v1 ServiceAccount resource
func (b *ServiceAccount) Reconcile(driver opsdk.Client) (sdk.Object, error) {
	err := b.buildConfigured()
	if err != nil {
		return nil, err
	}

	existing := &corev1.ServiceAccount{
		TypeMeta:   GetServiceAccountTypeMeta(),
		ObjectMeta: b.configured.ObjectMeta,
	}
	err = driver.Get(existing)
	if err != nil {
		return nil, errors.New("could not get existing")
	}

	if existing.ResourceVersion != "" {
		b.configured.ResourceVersion = existing.ResourceVersion
		return b.configured, nil
	}

	err = driver.Create(b.configured)
	if err == nil || k8serrors.IsAlreadyExists(err) {
		return b.configured, nil
	}

	return nil, err
}

// Build builds the service account kube api object
func (b *ServiceAccount) buildConfigured() error {
	serviceAccountName := fmt.Sprintf("%s-service-account", b.cluster.GetName())

	b.configured = &corev1.ServiceAccount{
		TypeMeta: GetServiceAccountTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: b.cluster.GetNamespace(),
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "quayio",
			},
		},
	}

	b.setOwner(asOwner(b.cluster))

	return nil
}

func (b *ServiceAccount) setOwner(owner metav1.OwnerReference) {
	b.configured.SetOwnerReferences(append(b.configured.GetOwnerReferences(), owner))
}
