package cassandracluster

import (
	"context"
	"log"

	"github.com/pantheon-systems/cassandra-operator-v0.0.1/pkg/backend/nodetool"
	databasev1alpha1 "github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var reconcilePeriod = 10

// Add creates a new CassandraCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	return &ReconcileCassandraCluster{
		client:        client,
		scheme:        mgr.GetScheme(),
		statusManager: controller.NewStatusManager(nodetoolDriver, client),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cassandracluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CassandraCluster
	err = c.Watch(&source.Kind{Type: &databasev1alpha1.CassandraCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CassandraCluster
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &databasev1alpha1.CassandraCluster{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCassandraCluster{}

// ReconcileCassandraCluster reconciles a CassandraCluster object
type ReconcileCassandraCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client         client.Client
	scheme         *runtime.Scheme
	statusManager  *controller.ClusterStatusManager
	nodetoolDriver *nodetool.Executor
}

// Reconcile reads that state of the cluster for a CassandraCluster object and makes changes based on the state read
// and what is in the CassandraCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCassandraCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling CassandraCluster %s/%s\n", request.Namespace, request.Name)

	// Fetch the CassandraCluster instance
	instance := &databasev1alpha1.CassandraCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if value, exists := instance.Annotations["database.pantheon.io/cassandra-operator-version"]; exists && value != opVersion.Version {
		return nil
	}
	// TODO: If there is already a pending update for a resource then
	// emit an error and do nothing for that update, leaving the actual
	// stored resource unchanged
	// NOTE: we could track and store the controllers...
	// update cluster status based on reality
	err := r.statusManager.Update(instance)
	if err != nil {
		return err
	}

	return controller.New(instance, r.client).Sync()
}
