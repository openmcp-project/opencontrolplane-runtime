package serviceprovider

import (
	"context"
	"time"

	"github.com/openmcp-project/opencontrolplane-runtime/pkg/serviceprovider/clusteraccess"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler implements any business logic required to manage API objects
type Reconciler[T API, C Config] interface {
	// CreateOrUpdate is called on every add or update event
	CreateOrUpdate(ctx context.Context, obj T, config C, clusters clusteraccess.ClusterContext) (ctrl.Result, error)
	// Delete is called on every delete event
	Delete(ctx context.Context, obj T, config C, clusters clusteraccess.ClusterContext) (ctrl.Result, error)
}

// API represents the end-user facing API type
type API interface {
	client.Object
	Status
	Finalizer() string
}

// Status represents the common status contract of API types
type Status interface {
	// GetStatus returns the status object
	GetStatus() any
	// GetConditions returns the status object
	GetConditions() *[]metav1.Condition
	// SetPhase sets Status.Phase
	SetPhase(string)
	// SetObservedGeneration sets Status.ObservedGeneration
	SetObservedGeneration(int64)
}

// Config represents the config for platform operators
// The Config is passed to the Reconciler to reconcile API objects
type Config interface {
	client.Object
	// PollInterval can be used to periodically requeue, preventing managed objects
	// from drifting on the target cluster.  Return 0 if not required.
	PollInterval() time.Duration
}
