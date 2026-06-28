package clusteraccess

import (
	"context"
	"testing"
	"time"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/opencontrolplane-runtime/pkg/clusterprovider"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/lib/clusteraccess/advanced"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	mcpID      = "mcp"
	workloadID = "workload"
)

func Test_advancedLocalAccessProvider_MCPCluster(t *testing.T) {
	tests := []struct {
		name     string
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
						clusterprovider.LocalAccessAnnotation: localAPIServer,
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
			fakeProvider := &fakeAdvancedClusterAccessReconciler{
				clusters:       map[string]*clusters.Cluster{mcpID: tt.cluster},
				accessRequests: map[string]*clustersv1alpha1.AccessRequest{mcpID: tt.ar},
			}
			localAccessProvider := NewLocalAdvancedClusterAccessReconciler(fakeProvider)
			got, gotErr := localAccessProvider.Access(context.Background(), reconcile.Request{}, mcpID)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Access() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Access() succeeded unexpectedly")
			}
			assert.Equal(t, tt.wantHost, got.RESTConfig().Host)
		})
	}
}

func Test_advancedLocalAccessProvider_WorkloadCluster(t *testing.T) {
	tests := []struct {
		name     string
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
						clusterprovider.LocalAccessAnnotation: localAPIServer,
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
			fakeProvider := &fakeAdvancedClusterAccessReconciler{
				clusters:       map[string]*clusters.Cluster{workloadID: tt.cluster},
				accessRequests: map[string]*clustersv1alpha1.AccessRequest{workloadID: tt.ar},
			}
			localAccessProvider := NewLocalAdvancedClusterAccessReconciler(fakeProvider)
			got, gotErr := localAccessProvider.Access(context.Background(), reconcile.Request{}, workloadID)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Access() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Access() succeeded unexpectedly")
			}
			assert.Equal(t, tt.wantHost, got.RESTConfig().Host)
		})
	}
}

var _ advanced.ClusterAccessReconciler = &fakeAdvancedClusterAccessReconciler{}

type fakeAdvancedClusterAccessReconciler struct {
	clusters       map[string]*clusters.Cluster
	accessRequests map[string]*clustersv1alpha1.AccessRequest
}

// Access implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) Access(_ context.Context, _ reconcile.Request, id string, _ ...any) (*clusters.Cluster, error) {
	return f.clusters[id], nil
}

// AccessRequest implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) AccessRequest(_ context.Context, _ reconcile.Request, id string, _ ...any) (*clustersv1alpha1.AccessRequest, error) {
	return f.accessRequests[id], nil
}

// ClusterRequest implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) ClusterRequest(_ context.Context, _ reconcile.Request, _ string, _ ...any) (*clustersv1alpha1.ClusterRequest, error) {
	panic("unimplemented")
}

// Cluster implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) Cluster(_ context.Context, _ reconcile.Request, _ string, _ ...any) (*clustersv1alpha1.Cluster, error) {
	panic("unimplemented")
}

// Reconcile implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) Reconcile(_ context.Context, _ reconcile.Request, _ ...any) (reconcile.Result, error) {
	panic("unimplemented")
}

// ReconcileDelete implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) ReconcileDelete(_ context.Context, _ reconcile.Request, _ ...any) (reconcile.Result, error) {
	panic("unimplemented")
}

// Register implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) Register(_ advanced.ClusterRegistration) advanced.ClusterAccessReconciler {
	panic("unimplemented")
}

// Unregister implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) Unregister(_ string) advanced.ClusterAccessReconciler {
	panic("unimplemented")
}

// Update implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) Update(_ string, _ ...advanced.ClusterRegistrationUpdate) error {
	panic("unimplemented")
}

// WithRetryInterval implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) WithRetryInterval(_ time.Duration) advanced.ClusterAccessReconciler {
	panic("unimplemented")
}

// WithManagedLabels implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) WithManagedLabels(_ advanced.ManagedLabelGenerator) advanced.ClusterAccessReconciler {
	panic("unimplemented")
}

// WithFakingCallback implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) WithFakingCallback(_ string, _ advanced.FakingCallback) advanced.ClusterAccessReconciler {
	panic("unimplemented")
}

// WithFakeClientGenerator implements [advanced.ClusterAccessReconciler].
func (f *fakeAdvancedClusterAccessReconciler) WithFakeClientGenerator(_ advanced.FakeClientGenerator) advanced.ClusterAccessReconciler {
	panic("unimplemented")
}
