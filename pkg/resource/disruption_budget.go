package resource

import (
	"errors"
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	opsdk "github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
func (b *PodDisruptionBudget) Reconcile(driver opsdk.Client) (sdk.Object, error) {
	b.buildDesired()

	existing := &policyv1beta1.PodDisruptionBudget{
		TypeMeta:   GetPodDisruptionBudgetTypeMeta(),
		ObjectMeta: b.desired.ObjectMeta,
	}
	err := driver.Get(existing)
	if err != nil {
		return nil, errors.New("could not get existing")
	}

	if existing.GetResourceVersion() != "" {
		b.desired.SetResourceVersion(existing.GetResourceVersion())
		err = driver.Update(b.desired)
	} else {
		err = driver.Create(b.desired)
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
	b.setOwner(asOwner(b.cluster))
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

func (b *PodDisruptionBudget) setOwner(owner metav1.OwnerReference) {
	b.desired.SetOwnerReferences(append(b.desired.GetOwnerReferences(), owner))
}
