package resource

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	opsdk "github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
)

// Reconciler reconciles the actual state and the desired state of an sdk.Object
type Reconciler interface {
	Reconcile(driver opsdk.Client) (sdk.Object, error)
}
