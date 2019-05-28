package resource

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles the actual state and the desired state of an sdk.Object
type Reconciler interface {
	Reconcile(ctx context.Context, driver client.Client) (runtime.Object, error)
}
