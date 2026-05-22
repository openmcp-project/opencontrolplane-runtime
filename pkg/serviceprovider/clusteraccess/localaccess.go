package clusteraccess

import (
	"context"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/api/common"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ clusteraccess.Reconciler = &localClusterAccessReconciler{}

// localClusterAccessReconciler is used for local debugging to adjust cluster client configs.
// Note that the builder methods have to be implemented to keep the pointer to the local impl
// instead of the wrapped reconciler.
type localClusterAccessReconciler struct {
	clusteraccess.Reconciler
}

// NewLocalAccessReconciler returns a local cluster access reconciler that wraps the given cluster access reconciler
func NewLocalAccessReconciler(car clusteraccess.Reconciler) clusteraccess.Reconciler {
	return &localClusterAccessReconciler{
		Reconciler: car,
	}
}

// MCPCluster implements [Provider].
func (s *localClusterAccessReconciler) MCPCluster(ctx context.Context, request reconcile.Request) (*clusters.Cluster, error) {
	cluster, err := s.Reconciler.MCPCluster(ctx, request)
	if err != nil {
		return cluster, err
	}
	ar, err := s.MCPAccessRequest(ctx, request)
	if err != nil {
		return cluster, err
	}
	// patch cluster client with annotation value
	return MustPatchClusterClient(ctx, ar, cluster), nil
}

// WorkloadCluster implements [Provider].
func (s *localClusterAccessReconciler) WorkloadCluster(ctx context.Context, request reconcile.Request) (*clusters.Cluster, error) {
	cluster, err := s.Reconciler.WorkloadCluster(ctx, request)
	if err != nil {
		return cluster, err
	}
	ar, err := s.WorkloadAccessRequest(ctx, request)
	if err != nil {
		return cluster, err
	}
	// patch cluster client with annotation value
	return MustPatchClusterClient(ctx, ar, cluster), nil
}

// SkipWorkloadCluster implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) SkipWorkloadCluster() clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.SkipWorkloadCluster()
	return s
}

// WithMCPPermissions implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithMCPPermissions(permissions []clustersv1alpha1.PermissionsRequest) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithMCPPermissions(permissions)
	return s
}

// WithMCPRoleRefs implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithMCPRoleRefs(roleRefs []common.RoleRef) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithMCPRoleRefs(roleRefs)
	return s
}

// WithMCPScheme implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithMCPScheme(scheme *runtime.Scheme) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithMCPScheme(scheme)
	return s
}

// WithRetryInterval implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithRetryInterval(interval time.Duration) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithRetryInterval(interval)
	return s
}

// WithWorkloadPermissions implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithWorkloadPermissions(permissions []clustersv1alpha1.PermissionsRequest) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithWorkloadPermissions(permissions)
	return s
}

// WithWorkloadRoleRefs implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithWorkloadRoleRefs(roleRefs []common.RoleRef) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithWorkloadRoleRefs(roleRefs)
	return s
}

// WithWorkloadScheme implements [clusteraccess.Reconciler].
func (s *localClusterAccessReconciler) WithWorkloadScheme(scheme *runtime.Scheme) clusteraccess.Reconciler {
	s.Reconciler = s.Reconciler.WithWorkloadScheme(scheme)
	return s
}

// sample AR
// apiVersion: clusters.openmcp.cloud/v1alpha1
// kind: AccessRequest
// metadata:
//
//	annotations:
//	  kind.clusters.openmcp.cloud/localhost: https://127.0.0.1:42827
const localAnnotationKey = "kind.clusters.openmcp.cloud/localhost"

// MustPatchClusterClient replaces the cluster client with the host value of the local AR annotation.
// If no local annotation is present then the original cluster of the wrapped reconciler is returned.
func MustPatchClusterClient(ctx context.Context, ar *clustersv1alpha1.AccessRequest, cluster *clusters.Cluster) *clusters.Cluster {
	annotations := ar.GetAnnotations()
	if annotations == nil {
		logf.FromContext(ctx).Info("debug access provider used but no annotations set")
		return cluster
	}

	value, exists := annotations[localAnnotationKey]
	if !exists {
		logf.FromContext(ctx).Info("debug access provider used but local annotation key not found", "key", localAnnotationKey)
		return cluster
	}

	if value == "" {
		logf.FromContext(ctx).Info("local annotation present but empty, skipping patch")
		return cluster
	}
	cfg := *cluster.RESTConfig()
	cfg.Host = value
	cluster.WithRESTConfig(&cfg)

	// re-init client
	if err := cluster.InitializeClient(cluster.Client().Scheme()); err != nil {
		panic(err)
	}
	return cluster
}
