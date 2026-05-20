package serviceprovider

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

// Reconciler implements any business logic required to manage API objects
type Reconciler[T API, PC ProviderConfig] interface {
	// CreateOrUpdate is called on every add or update event
	CreateOrUpdate(ctx context.Context, obj T, pc PC, clusters ClusterContext) (ctrl.Result, error)
	// Delete is called on every delete event
	Delete(ctx context.Context, obj T, pc PC, clusters ClusterContext) (ctrl.Result, error)
}
