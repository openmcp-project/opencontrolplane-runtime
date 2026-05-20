package serviceprovider

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/controller-utils/pkg/controller"
)

// ConfigReconciler notifies the service provider about provider config updates
// through a shared update channel. Any provider config change results in a reconcile request
// for every existing service provider api object.
type ConfigReconciler[T ProviderConfig] struct {
	platformCluster       *clusters.Cluster
	providerUpdateChannel chan event.GenericEvent
	providerName          string
	emptyObj              func() T
}

// NewProviderConfigReconciler creates a new provider PCReconciler instance.
func NewProviderConfigReconciler[T ProviderConfig](providerName string, emptyObj func() T) *ConfigReconciler[T] {
	return &ConfigReconciler[T]{
		providerName: providerName,
		emptyObj:     emptyObj,
	}
}

// WithPlatformCluster sets the platform cluster.
func (r *ConfigReconciler[T]) WithPlatformCluster(c *clusters.Cluster) *ConfigReconciler[T] {
	r.platformCluster = c
	return r
}

// WithUpdateChannel sets the channel to send config changes.
func (r *ConfigReconciler[T]) WithUpdateChannel(c chan event.GenericEvent) *ConfigReconciler[T] {
	r.providerUpdateChannel = c
	return r
}

// Reconcile acts as a sender to notify receivers about provider config changes .
func (r *ConfigReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.FromContext(ctx).Info("reconcile provider config")
	obj := r.emptyObj()
	notify := event.GenericEvent{}
	if err := r.platformCluster.Client().Get(ctx, req.NamespacedName, obj); err != nil {
		r.providerUpdateChannel <- notify
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !obj.GetDeletionTimestamp().IsZero() {
		r.providerUpdateChannel <- notify
		return ctrl.Result{}, nil
	}
	notify.Object = obj.DeepCopyObject().(T)
	r.providerUpdateChannel <- notify
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReconciler[T]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(source.Kind(
			r.platformCluster.Cluster().GetCache(),
			r.emptyObj(),
			&handler.TypedEnqueueRequestForObject[T]{},
			controller.ToTypedPredicate[T](controller.ExactNamePredicate(r.providerName, "")),
		)).
		Named("providerconfig").
		Complete(r)
}
