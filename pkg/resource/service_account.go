package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
func (b *ServiceAccount) Reconcile(ctx context.Context, driver client.Client) (runtime.Object, error) {
	err := b.buildConfigured()
	if err != nil {
		return nil, err
	}

	namespacedName := types.NamespacedName{
		Namespace: b.cluster.GetNamespace(),
		Name:      b.cluster.GetName(),
	}

	existing := &corev1.ServiceAccount{}
	err = driver.Get(ctx, namespacedName, existing)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, errors.New("could not get existing")
	}

	if existing.ResourceVersion != "" {
		b.configured.ResourceVersion = existing.ResourceVersion
		return b.configured, nil
	}

	err = driver.Create(ctx, b.configured)
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

	controllerutil.SetControllerReference(b.cluster, b.configured, scheme.Scheme)

	return nil
}
