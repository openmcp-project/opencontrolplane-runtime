package clusteraccess

import (
	"context"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess/advanced"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ advanced.ClusterAccessReconciler = &localAdvancedClusterAccessReconciler{}

// localAdvancedClusterAccessReconciler is used for local debugging to adjust cluster client configs.
// Note that the builder methods have to be implemented to keep the pointer to the local impl
// instead of the wrapped reconciler.
type localAdvancedClusterAccessReconciler struct {
	advanced.ClusterAccessReconciler
}

// NewLocalAdvancedClusterAccessReconciler returns a local advanced cluster access reconciler that wraps the given advanced cluster access reconciler.
func NewLocalAdvancedClusterAccessReconciler(car advanced.ClusterAccessReconciler) advanced.ClusterAccessReconciler {
	return &localAdvancedClusterAccessReconciler{
		ClusterAccessReconciler: car,
	}
}

// Access implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) Access(ctx context.Context, request reconcile.Request, id string, additionalData ...any) (*clusters.Cluster, error) {
	cluster, err := s.ClusterAccessReconciler.Access(ctx, request, id, additionalData...)
	if err != nil {
		return cluster, err
	}
	ar, err := s.AccessRequest(ctx, request, id, additionalData...)
	if err != nil {
		return cluster, err
	}
	return MustPatchClusterClient(ctx, ar, cluster), nil
}

// Register implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) Register(reg advanced.ClusterRegistration) advanced.ClusterAccessReconciler {
	s.ClusterAccessReconciler = s.ClusterAccessReconciler.Register(reg)
	return s
}

// Unregister implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) Unregister(id string) advanced.ClusterAccessReconciler {
	s.ClusterAccessReconciler = s.ClusterAccessReconciler.Unregister(id)
	return s
}

// WithRetryInterval implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) WithRetryInterval(interval time.Duration) advanced.ClusterAccessReconciler {
	s.ClusterAccessReconciler = s.ClusterAccessReconciler.WithRetryInterval(interval)
	return s
}

// WithManagedLabels implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) WithManagedLabels(gen advanced.ManagedLabelGenerator) advanced.ClusterAccessReconciler {
	s.ClusterAccessReconciler = s.ClusterAccessReconciler.WithManagedLabels(gen)
	return s
}

// WithFakingCallback implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) WithFakingCallback(key string, callback advanced.FakingCallback) advanced.ClusterAccessReconciler {
	s.ClusterAccessReconciler = s.ClusterAccessReconciler.WithFakingCallback(key, callback)
	return s
}

// WithFakeClientGenerator implements [advanced.ClusterAccessReconciler].
func (s *localAdvancedClusterAccessReconciler) WithFakeClientGenerator(f advanced.FakeClientGenerator) advanced.ClusterAccessReconciler {
	s.ClusterAccessReconciler = s.ClusterAccessReconciler.WithFakeClientGenerator(f)
	return s
}
