package stub

import (
	"context"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	opVersion "github.com/pantheon-systems/cassandra-operator/version"

	opsdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
)

// NewHandler creates a new handler for the cassandra cluster operator
func NewHandler(k8sDriver k8s.Client, nodetoolDriver *nodetool.Executor) opsdk.Handler {
	statusManager := controller.NewStatusManager(nodetoolDriver, k8sDriver)
	return &Handler{
		k8sDriver:      k8sDriver,
		statusManager:  statusManager,
		nodetoolDriver: nodetoolDriver,
	}
}

// Handler is the cassandra cluster operator structure for handling events
type Handler struct {
	k8sDriver      k8s.Client
	statusManager  *controller.ClusterStatusManager
	nodetoolDriver *nodetool.Executor
}

// Handle takes events and dispatches to synchronization code
func (h *Handler) Handle(ctx context.Context, event opsdk.Event) error {
	var err error
	switch o := event.Object.(type) {
	case *v1alpha1.CassandraCluster:
		err = h.handleCassandraClusterEvent(o, event.Deleted)
	case *corev1.Pod:
		err = h.handlePodEvent(o, event.Deleted)
	}
	return err
}

func (h *Handler) handleCassandraClusterEvent(o *v1alpha1.CassandraCluster, deleted bool) error {
	if value, exists := o.Annotations["database.panth.io/cassandra-operator-version"]; exists && value != opVersion.Version {
		return nil
	}
	// TODO: If there is already a pending update for a resource then
	// emit an error and do nothing for that update, leaving the actual
	// stored resource unchanged
	// NOTE: we could track and store the controllers...
	if !deleted {
		// update cluster status based on reality
		err := h.statusManager.Update(o)
		if err != nil {
			return err
		}

		return controller.New(o, h.k8sDriver).Sync()
	}

	return nil
}

func (h *Handler) handlePodEvent(o *corev1.Pod, deleted bool) error {
	if o.Annotations["disable-pod-finalizer"] == "true" {
		return nil
	}

	podFinalizerCtrl := controller.NewPodFinalizerController(h.k8sDriver, h.nodetoolDriver)
	err := podFinalizerCtrl.Converge(o)
	if err != nil {
		return err
	}

	return podFinalizerCtrl.Process(o)
}
