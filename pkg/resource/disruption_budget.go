package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// PodDisruptionBudget class that builds a PodDisruptionBudiget and reconciles the
// actual state to desired state
type PodDisruptionBudget struct {
	cluster *v1alpha1.CassandraCluster
	desired *policyv1beta1.PodDisruptionBudget
}

// NewPodDisruptionBudget creates a new PodDIsruptionBudget
func NewPodDisruptionBudget(cc *v1alpha1.CassandraCluster) *PodDisruptionBudget {
	return &PodDisruptionBudget{
		cluster: cc,
	}
}

// Reconcile the PodDisruptionBudget's actual state with desired
func (b *PodDisruptionBudget) Reconcile(ctx context.Context, driver client.Client) (runtime.Object, error) {
	b.buildDesired()

	namespacedName := types.NamespacedName{
		Namespace: b.desired.GetNamespace(),
		Name:      b.desired.GetName(),
	}

	existing := &policyv1beta1.PodDisruptionBudget{}
	err := driver.Get(ctx, namespacedName, existing)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, errors.New("could not get existing")
	}

	if existing.GetResourceVersion() != "" {
		b.desired.SetResourceVersion(existing.GetResourceVersion())
		err = driver.Update(ctx, b.desired)
	} else {
		err = driver.Create(ctx, b.desired)
	}

	if err != nil {
		return nil, err
	}
	return b.desired, nil
}

func (b *PodDisruptionBudget) buildDesired() {
	two := intstr.Parse("2")

	b.desired = &policyv1beta1.PodDisruptionBudget{
		TypeMeta: GetPodDisruptionBudgetTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cassandra", b.cluster.GetName()),
			Namespace: b.cluster.GetNamespace(),
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			MinAvailable: &two,
			Selector:     &metav1.LabelSelector{},
		},
	}

	b.buildSelector()
	controllerutil.SetControllerReference(b.cluster, b.desired, scheme.Scheme)
}

func (b *PodDisruptionBudget) buildSelector() {
	b.desired.Spec.Selector.MatchLabels = map[string]string{
		"cluster": b.cluster.GetName(),
		"state":   "serving",
	}

	if appName, ok := b.cluster.GetLabels()["app"]; ok {
		b.desired.Spec.Selector.MatchLabels["app"] = appName
	}
}
