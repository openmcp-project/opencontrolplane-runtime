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
type ConfigReconciler[T Config] struct {
	platformCluster       *clusters.Cluster
	providerUpdateChannel chan event.GenericEvent
	providerName          string
	emptyObj              func() T
}

// ConfigReconcilerBuilder enables building valid ConfigReconcilers.
type ConfigReconcilerBuilder[T Config] struct {
	configReconciler ConfigReconciler[T]
}

// NewConfigReconcilerBuilder creates a builder.
func NewConfigReconcilerBuilder[T Config]() *ConfigReconcilerBuilder[T] {
	return &ConfigReconcilerBuilder[T]{
		configReconciler: ConfigReconciler[T]{},
	}
}

// MustBuild validates every required field has been set and returns the ConfigReconciler.
func (b *ConfigReconcilerBuilder[T]) MustBuild() *ConfigReconciler[T] {
	// validate required fields
	if b.configReconciler.emptyObj == nil {
		panic("empty object provider is required")
	}
	if b.configReconciler.platformCluster == nil {
		panic("platform cluster is required")
	}
	if b.configReconciler.providerUpdateChannel == nil {
		panic("update channel is required")
	}
	if b.configReconciler.providerName == "" {
		panic("provider name is required")
	}
	return &b.configReconciler
}

// EmptyObjectProvider sets the empty object function required for concrete type processing.
func (b *ConfigReconcilerBuilder[T]) EmptyObjectProvider(emptyObj func() T) *ConfigReconcilerBuilder[T] {
	b.configReconciler.emptyObj = emptyObj
	return b
}

// PlatformCluster sets the platform cluster.
func (b *ConfigReconcilerBuilder[T]) PlatformCluster(c *clusters.Cluster) *ConfigReconcilerBuilder[T] {
	b.configReconciler.platformCluster = c
	return b
}

// ProviderName sets the provider name.
func (b *ConfigReconcilerBuilder[T]) ProviderName(name string) *ConfigReconcilerBuilder[T] {
	b.configReconciler.providerName = name
	return b
}

// UpdateChannel sets the channel to send config changes.
func (b *ConfigReconcilerBuilder[T]) UpdateChannel(c chan event.GenericEvent) *ConfigReconcilerBuilder[T] {
	b.configReconciler.providerUpdateChannel = c
	return b
}

// Reconcile acts as a sender to notify receivers about provider config changes.
func (b *ConfigReconciler[T]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.FromContext(ctx).Info("reconcile provider config")
	obj := b.emptyObj()
	notify := event.GenericEvent{}
	if err := b.platformCluster.Client().Get(ctx, req.NamespacedName, obj); err != nil {
		b.providerUpdateChannel <- notify
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !obj.GetDeletionTimestamp().IsZero() {
		b.providerUpdateChannel <- notify
		return ctrl.Result{}, nil
	}
	notify.Object = obj.DeepCopyObject().(T)
	b.providerUpdateChannel <- notify
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (b *ConfigReconciler[T]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(source.Kind(
			b.platformCluster.Cluster().GetCache(),
			b.emptyObj(),
			&handler.TypedEnqueueRequestForObject[T]{},
			controller.ToTypedPredicate[T](controller.ExactNamePredicate(b.providerName, "")),
		)).
		Named("providerconfig").
		Complete(b)
}
