package cassandracluster

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	databasev1alpha1 "github.com/pantheon-systems/cassandra-operator/pkg/apis/database/v1alpha1"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"github.com/pantheon-systems/cassandra-operator/version"
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

const (
	reconcilePeriod = 30 * time.Second
)

// Add creates a new CassandraCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, mgr manager.Manager) error {
	k8sPodExecutor := k8s.NewPodExecutor(mgr.GetConfig())
	nodetoolClient := nodetool.NewExecutor(k8sPodExecutor)

	k8sClient := mgr.GetClient()
	statusManager := NewStatusManager(ctx, nodetoolClient, k8sClient)

	return add(mgr, newReconciler(ctx, mgr, nodetoolClient, statusManager))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(ctx context.Context, mgr manager.Manager, nodetoolClient *nodetool.Executor, statusMgr *ClusterStatusManager) reconcile.Reconciler {
	return &ReconcileCassandraCluster{
		ctx:            ctx,
		client:         mgr.GetClient(),
		scheme:         mgr.GetScheme(),
		nodetoolClient: nodetoolClient,
		statusMgr:      statusMgr,
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
	client client.Client
	scheme *runtime.Scheme

	statusMgr      *ClusterStatusManager
	nodetoolClient *nodetool.Executor
	ctx            context.Context
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
	err := r.client.Get(r.ctx, request.NamespacedName, instance)
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

	if value, exists := instance.Annotations["database.panth.io/cassandra-operator-version"]; exists && value != version.Version {
		return reconcile.Result{}, nil
	}

	// update cluster status based on reality
	// this sets the status
	err = r.statusMgr.Update(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// act on that set status
	switch instance.Status.Phase {
	case "":
		instance.Annotations["database.panth.io/cassandra-operator-version"] = version.Version
		err := r.client.Update(r.ctx, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	case v1alpha1.ClusterPhaseInitial:
		logrus.Debugf("ClusterPhaseInitial for cluster: %s", instance.GetName())
		// initial is the default phase when the cluster object is created
		err := r.validateConfigmaps()
		if err != nil {
			return reconcile.Result{}, err
		}
		// Vault -> maybe a plugin interface here for the OSS project
		err = r.validateSecrets()
		if err != nil {
			return reconcile.Result{}, err
		}
	case v1alpha1.ClusterPhaseFailed:
		return reconcile.Result{}, fmt.Errorf("provisioning cluster has failed")
	default:
		if instance.Status.Provisioning() {
			logrus.Debugf("Nodes are provisioning for cluster %s, no-op and wait", instance.GetName())
			return reconcile.Result{RequeueAfter: reconcilePeriod}, nil
		}

		if instance.Status.NodesInTransit() {
			logrus.Debugf("Nodes are in motion for cluster %s, no-op and wait", instance.GetName())
			return reconcile.Result{RequeueAfter: reconcilePeriod}, nil
		}
	}

	err = r.reconcile(instance)
	return reconcile.Result{}, err
}

func (r *ReconcileCassandraCluster) validateSecrets() error {
	return nil
}

func (r *ReconcileCassandraCluster) validateConfigmaps() error {
	return nil
}
