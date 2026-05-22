package clusteraccess

import (
	"context"
	"testing"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/api/common"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	inclusterAPIServer = "https://10.96.0.1:6443"
	localAPIServer     = "https://127.0.0.1:12345"
)

func Test_localAccessProvider_MCPCluster(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		ar       *clustersv1alpha1.AccessRequest
		cluster  *clusters.Cluster
		wantHost string
		wantErr  bool
	}{
		{
			name: "local annotation results in local client config",
			ar: &clustersv1alpha1.AccessRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcp-access",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						localAnnotationKey: localAPIServer,
					},
				},
			},
			cluster:  createFakeCluster().WithRESTConfig(&rest.Config{Host: inclusterAPIServer}),
			wantHost: localAPIServer,
			wantErr:  false,
		},
		{
			name: "no local annotation results in original cluster client config",
			ar: &clustersv1alpha1.AccessRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mcp-access",
					Namespace: metav1.NamespaceDefault,
				},
			},
			cluster:  createFakeCluster().WithRESTConfig(&rest.Config{Host: inclusterAPIServer}),
			wantHost: inclusterAPIServer,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeProvider := &fakeClusterAccessReconciler{
				ManagedControlPlane:   tt.cluster,
				ManagedControlPlaneAR: tt.ar,
			}
			localAccessProvider := NewLocalAccessReconciler(fakeProvider)
			got, gotErr := localAccessProvider.MCPCluster(context.Background(), reconcile.Request{})
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("MCPCluster() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("MCPCluster() succeeded unexpectedly")
			}
			assert.Equal(t, tt.wantHost, got.RESTConfig().Host)
		})
	}
}

func Test_localAccessProvider_WorkloadCluster(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		ar       *clustersv1alpha1.AccessRequest
		cluster  *clusters.Cluster
		wantHost string
		wantErr  bool
	}{
		{
			name: "local annotation results in local client config",
			ar: &clustersv1alpha1.AccessRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "workload-access",
					Namespace: metav1.NamespaceDefault,
					Annotations: map[string]string{
						localAnnotationKey: localAPIServer,
					},
				},
			},
			cluster:  createFakeCluster().WithRESTConfig(&rest.Config{Host: inclusterAPIServer}),
			wantHost: localAPIServer,
			wantErr:  false,
		},
		{
			name: "no local annotation results in original cluster client config",
			ar: &clustersv1alpha1.AccessRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "workload-access",
					Namespace: metav1.NamespaceDefault,
				},
			},
			cluster:  createFakeCluster().WithRESTConfig(&rest.Config{Host: inclusterAPIServer}),
			wantHost: inclusterAPIServer,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeProvider := &fakeClusterAccessReconciler{
				Workload:   tt.cluster,
				WorkloadAR: tt.ar,
			}
			localAccessProvider := NewLocalAccessReconciler(fakeProvider)
			got, gotErr := localAccessProvider.WorkloadCluster(context.Background(), reconcile.Request{})
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("WorkloadCluster() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("WorkloadCluster() succeeded unexpectedly")
			}
			assert.Equal(t, tt.wantHost, got.RESTConfig().Host)
		})
	}
}

var _ clusteraccess.Reconciler = &fakeClusterAccessReconciler{}

type fakeClusterAccessReconciler struct {
	ManagedControlPlane   *clusters.Cluster
	ManagedControlPlaneAR *clustersv1alpha1.AccessRequest
	Workload              *clusters.Cluster
	WorkloadAR            *clustersv1alpha1.AccessRequest
}

// MCPAccessRequest implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) MCPAccessRequest(ctx context.Context, request reconcile.Request) (*clustersv1alpha1.AccessRequest, error) {
	return f.ManagedControlPlaneAR, nil
}

// MCPCluster implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) MCPCluster(ctx context.Context, request reconcile.Request) (*clusters.Cluster, error) {
	return f.ManagedControlPlane, nil
}

// Reconcile implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	panic("unimplemented")
}

// ReconcileDelete implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) ReconcileDelete(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	panic("unimplemented")
}

// SkipWorkloadCluster implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) SkipWorkloadCluster() clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithMCPPermissions implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithMCPPermissions(permissions []clustersv1alpha1.PermissionsRequest) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithMCPRoleRefs implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithMCPRoleRefs(roleRefs []common.RoleRef) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithMCPScheme implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithMCPScheme(scheme *runtime.Scheme) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithRetryInterval implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithRetryInterval(interval time.Duration) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithWorkloadPermissions implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithWorkloadPermissions(permissions []clustersv1alpha1.PermissionsRequest) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithWorkloadRoleRefs implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithWorkloadRoleRefs(roleRefs []common.RoleRef) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WithWorkloadScheme implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WithWorkloadScheme(scheme *runtime.Scheme) clusteraccess.Reconciler {
	panic("unimplemented")
}

// WorkloadAccessRequest implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WorkloadAccessRequest(ctx context.Context, request reconcile.Request) (*clustersv1alpha1.AccessRequest, error) {
	return f.WorkloadAR, nil
}

// WorkloadCluster implements [clusteraccess.Reconciler].
func (f *fakeClusterAccessReconciler) WorkloadCluster(ctx context.Context, request reconcile.Request) (*clusters.Cluster, error) {
	return f.Workload, nil
}

func createFakeCluster() *clusters.Cluster {
	return clusters.NewTestClusterFromClient("fakeCluster", fake.NewClientBuilder().Build())
}
