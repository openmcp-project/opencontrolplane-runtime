package clusteraccess

import (
	"context"
	"fmt"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess/advanced"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ advanced.ClusterAccessReconciler = &localAdvancedClusterAccessReconciler{}

// localAdvancedClusterAccessReconciler is used for local debugging to adjust cluster client configs.
// Note that the builder methods have to be implemented to keep the pointer to the local impl
// instead of the wrapped reconciler.
type localAdvancedClusterAccessReconciler struct {
	advanced.ClusterAccessReconciler
	withWorkload bool
}

// NewLocalAdvancedClusterAccessReconciler returns a local advanced cluster access reconciler that wraps the given advanced cluster access reconciler.
// Set withWorkload to true when the service provider deploys to a workload cluster
func NewLocalAdvancedClusterAccessReconciler(car advanced.ClusterAccessReconciler, withWorkload bool) advanced.ClusterAccessReconciler {
	return &localAdvancedClusterAccessReconciler{
		ClusterAccessReconciler: car,
		withWorkload:            withWorkload,
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
	// Always patch the cluster client with the host value of the local AR annotation so that the service provider process can connect.
	cluster = MustPatchClusterClient(ctx, ar, cluster)

	// If the service provider is using a workload cluster we additionally have to override the MCPs rest.Config.Host to the Docker-network address fetched from the "apiserver-internal" endpoint of the Cluster.
	// Using this endpoint, the pod running on the workload cluster can reach the MCP API server if injected as KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT env vars.
	// If we would not override it the rest.Config.Host would point to localhost.
	// Warning: This does not affect the cluster client as we only initialize it in MustPatchClusterClient. As a result the rest.Config points to a different host than the cluster client!
	if id == MCPClusterID && s.withWorkload && cluster.HasRESTConfig() {
		mcpCluster, err := s.Cluster(ctx, request, id, additionalData...)
		if err != nil {
			return cluster, err
		}
		if mcpCluster == nil {
			return cluster, fmt.Errorf("mcp cluster not found")
		}
		internalURL, ok := mcpCluster.Status.Endpoints.Get(clustersv1alpha1.APISERVER_ENDPOINT_INTERNAL)
		if !ok {
			return cluster, fmt.Errorf("%s endpoint not found", clustersv1alpha1.APISERVER_ENDPOINT_INTERNAL)
		}
		cfg := *cluster.RESTConfig()
		cfg.Host = internalURL
		cluster.WithRESTConfig(&cfg)
	}
	return cluster, nil
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
