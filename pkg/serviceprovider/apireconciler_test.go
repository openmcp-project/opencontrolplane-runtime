package serviceprovider

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/zapr"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/opencontrolplane-runtime/pkg/serviceprovider/clusteraccess"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/api/common"
	apiconst "github.com/openmcp-project/openmcp-operator/api/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespaceName                = "test-namespace"
	testObjectName                   = "test-name"
	testObjectNameNotFound           = "notfound"
	testObjectNameClusterAccessError = "clusteraccesserror"

	testMCPName       = "mcp-name"
	testMCPKubeconfig = "mcp-kubeconfig"

	testWorkloadName       = "workload-name"
	testWorkloadKubeconfig = "workload-kubeconfig"
)

func TestAPIReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		apiObj             API
		providerConfig     *fakeProviderConfigImpl
		req                ctrl.Request
		want               ctrl.Result
		wantStatusPhase    string
		wantReconciliation bool
		wantErr            bool
	}{
		{
			name: "CreateOrUpdate ok -> requeue with pc poll interval",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			providerConfig: &fakeProviderConfigImpl{
				FakePollInterval: time.Hour,
			},
			want: ctrl.Result{
				RequeueAfter: time.Hour,
			},
			wantStatusPhase:    StatusPhaseReady,
			wantReconciliation: true,
			wantErr:            false,
		},
		{
			name: "CreateOrUpdate error -> error and status update",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			providerConfig: &fakeProviderConfigImpl{
				FakePollInterval: time.Hour,
			},
			want:               ctrl.Result{},
			wantStatusPhase:    StatusPhaseProgressing,
			wantReconciliation: true,
			wantErr:            true,
		},
		{
			name: "Delete ok -> requeue with pc poll interval",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"string"},
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			providerConfig: &fakeProviderConfigImpl{
				FakePollInterval: time.Hour,
			},
			want: ctrl.Result{
				RequeueAfter: time.Hour,
			},
			wantStatusPhase:    StatusPhaseTerminating,
			wantReconciliation: true,
			wantErr:            false,
		},
		{
			name: "Delete error -> error and status update",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"string"},
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			providerConfig: &fakeProviderConfigImpl{
				FakePollInterval: time.Hour,
			},
			want:               ctrl.Result{},
			wantStatusPhase:    StatusPhaseTerminating,
			wantReconciliation: true,
			wantErr:            true,
		},
		{
			name: "api obj not found -> do not requeue",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"string"},
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectNameNotFound,
					Namespace: testNamespaceName,
				},
			},
			providerConfig: &fakeProviderConfigImpl{
				FakePollInterval: time.Hour,
			},
			want:               ctrl.Result{},
			wantStatusPhase:    "",
			wantReconciliation: false,
			wantErr:            false,
		},
		{
			name: "provider config not found -> error",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			want:               ctrl.Result{},
			wantStatusPhase:    StatusPhaseProgressing,
			wantReconciliation: false,
			wantErr:            true,
		},
		{
			name: "Operation annotation ignore -> no reconciliation, no requeue",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectName,
					Namespace: testNamespaceName,
					Annotations: map[string]string{
						apiconst.OperationAnnotation: apiconst.OperationAnnotationValueIgnore,
					},
				},
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectName,
					Namespace: testNamespaceName,
				},
			},
			providerConfig:     &fakeProviderConfigImpl{},
			want:               ctrl.Result{},
			wantReconciliation: false,
			wantErr:            false,
		},
		{
			name: "cluster access reconciler fails -> error and status update",
			apiObj: &fakeApiImpl{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testObjectNameClusterAccessError,
					Namespace: testNamespaceName,
				},
			},
			providerConfig: &fakeProviderConfigImpl{
				FakePollInterval: time.Hour,
			},
			req: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      testObjectNameClusterAccessError,
					Namespace: testNamespaceName,
				},
			},
			want:               ctrl.Result{},
			wantStatusPhase:    StatusPhaseProgressing,
			wantReconciliation: true,
			wantErr:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onboardingCluster := createFakeCluster(t, "onboarding", tt.apiObj)
			platformCluster := createFakeCluster(t, "platform")
			mockReconciler := &MockServiceProviderReconciler{
				wantError: tt.wantErr,
			}
			builder := NewAPIReconcilerBuilder[*fakeApiImpl, *fakeProviderConfigImpl]().
				EmptyObjectProvider(func() *fakeApiImpl { return &fakeApiImpl{} }).
				OnboardingCluster(onboardingCluster).
				PlatformCluster(platformCluster).
				ClusterAccessReconciler(FakeClusterAccessProvider{
					ManagedControlPlane: createFakeCluster(t, testMCPName),
					ManagedControlPlaneAR: &clustersv1alpha1.AccessRequest{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testMCPName,
							Namespace: testNamespaceName,
						},
						Status: clustersv1alpha1.AccessRequestStatus{
							SecretRef: &common.LocalObjectReference{
								Name: testMCPKubeconfig,
							},
						},
					},
					Workload: createFakeCluster(t, testWorkloadName),
					WorkloadAR: &clustersv1alpha1.AccessRequest{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testWorkloadName,
							Namespace: testNamespaceName,
						},
						Status: clustersv1alpha1.AccessRequestStatus{
							SecretRef: &common.LocalObjectReference{
								Name: testWorkloadKubeconfig,
							},
						},
					},
				}).
				Reconciler(mockReconciler).
				WorkloadCluster(true)
			if tt.providerConfig != nil {
				builder.ProviderConfig(tt.providerConfig)
			}
			r := builder.MustBuild()
			got, gotErr := r.Reconcile(context.Background(), tt.req)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Reconcile() failed: %v", gotErr)
				}
				assertStatusUpdate(t, onboardingCluster.Client(), tt.req, tt.wantStatusPhase)
				return
			}
			if tt.wantErr {
				t.Fatal("Reconcile() succeeded unexpectedly")
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantReconciliation, mockReconciler.createOrUpdateCalled || mockReconciler.deleteCalled)

			if !tt.wantReconciliation {
				assert.False(t, mockReconciler.createOrUpdateCalled)
				assert.False(t, mockReconciler.deleteCalled)
				assert.Nil(t, mockReconciler.apiObj)
				assert.Nil(t, mockReconciler.config)
				assert.Empty(t, mockReconciler.clusterContext.MCPAccessSecretKey)
				assert.Empty(t, mockReconciler.clusterContext.WorkloadAccessSecretKey)
				return
			}

			// assert that the generic reconciler delegates objects to the target reconciler as expected
			assert.Equal(t, client.ObjectKeyFromObject(tt.apiObj), client.ObjectKeyFromObject(mockReconciler.apiObj))
			assert.Equal(t, client.ObjectKeyFromObject(tt.providerConfig), client.ObjectKeyFromObject(mockReconciler.config))
			assert.Equal(t, client.ObjectKey{
				Namespace: tt.req.Namespace,
				Name:      testMCPKubeconfig,
			}, mockReconciler.clusterContext.MCPAccessSecretKey)
			assert.Equal(t, client.ObjectKey{
				Namespace: tt.req.Namespace,
				Name:      testWorkloadKubeconfig,
			}, mockReconciler.clusterContext.WorkloadAccessSecretKey)
			assertStatusUpdate(t, onboardingCluster.Client(), tt.req, tt.wantStatusPhase)
		})
	}
}

func assertStatusUpdate(t *testing.T, c client.Client, req ctrl.Request, wantStatusPhase string) {
	t.Helper()
	obj := &fakeApiImpl{}
	obj.SetName(req.Name)
	obj.SetNamespace(req.Namespace)
	require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(obj), obj))
	status, ok := obj.GetStatus().(common.Status)
	require.True(t, ok)
	assert.Equal(t, wantStatusPhase, status.Phase)
}

var _ clusteraccess.Provider = FakeClusterAccessProvider{}
var _ Reconciler[*fakeApiImpl, *fakeProviderConfigImpl] = &MockServiceProviderReconciler{}

type MockServiceProviderReconciler struct {
	apiObj               API
	config               Config
	clusterContext       clusteraccess.ClusterContext
	createOrUpdateCalled bool
	deleteCalled         bool
	wantError            bool
}

// CreateOrUpdate implements [Reconciler].
func (f *MockServiceProviderReconciler) CreateOrUpdate(_ context.Context, obj *fakeApiImpl, pc *fakeProviderConfigImpl, cc clusteraccess.ClusterContext) (ctrl.Result, error) {
	f.apiObj = obj
	f.config = pc
	f.clusterContext = cc
	f.createOrUpdateCalled = true
	if f.wantError {
		StatusProgressing(obj, reasonReconcileError, "test error requested")
		return reconcile.Result{}, errors.New("createOrUpdate failed")
	}
	StatusReady(obj)
	return reconcile.Result{}, nil
}

// Delete implements [Reconciler].
func (f *MockServiceProviderReconciler) Delete(_ context.Context, obj *fakeApiImpl, pc *fakeProviderConfigImpl, cc clusteraccess.ClusterContext) (ctrl.Result, error) {
	f.apiObj = obj
	f.config = pc
	f.clusterContext = cc
	f.deleteCalled = true
	StatusTerminating(obj)
	if f.wantError {
		return reconcile.Result{}, errors.New("delete failed")
	}
	return reconcile.Result{}, nil
}

type FakeClusterAccessProvider struct {
	ManagedControlPlane   *clusters.Cluster
	ManagedControlPlaneAR *clustersv1alpha1.AccessRequest
	Workload              *clusters.Cluster
	WorkloadAR            *clustersv1alpha1.AccessRequest
}

// MCPAccessRequest implements [ClusterAccessProvider].
func (f FakeClusterAccessProvider) MCPAccessRequest(ctx context.Context, request reconcile.Request) (*clustersv1alpha1.AccessRequest, error) {
	return f.ManagedControlPlaneAR, nil
}

// MCPCluster implements [ClusterAccessProvider].
func (f FakeClusterAccessProvider) MCPCluster(ctx context.Context, request reconcile.Request) (*clusters.Cluster, error) {
	return f.ManagedControlPlane, nil
}

// Reconcile implements [ClusterAccessProvider].
func (f FakeClusterAccessProvider) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	if request.Name == testObjectNameClusterAccessError {
		return reconcile.Result{}, errors.New("cluster access reconcile failed")
	}
	return reconcile.Result{}, nil
}

// ReconcileDelete implements [ClusterAccessProvider].
func (f FakeClusterAccessProvider) ReconcileDelete(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Fake waiting for cluster acccess deletion.
	// This prevents finalizer removal in the delete case and the fake client from losing the not yet persisted terminating state.
	return reconcile.Result{
		RequeueAfter: time.Hour,
	}, nil
}

// WorkloadAccessRequest implements [ClusterAccessProvider].
func (f FakeClusterAccessProvider) WorkloadAccessRequest(ctx context.Context, request reconcile.Request) (*clustersv1alpha1.AccessRequest, error) {
	return f.WorkloadAR, nil
}

// WorkloadCluster implements [ClusterAccessProvider].
func (f FakeClusterAccessProvider) WorkloadCluster(ctx context.Context, request reconcile.Request) (*clusters.Cluster, error) {
	return f.Workload, nil
}

var testGV = schema.GroupVersion{Group: "openmcp.test", Version: "v1"}

func createFakeCluster(t *testing.T, id string, clusterObjects ...client.Object) *clusters.Cluster {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = apiextv1.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)
	scheme.AddKnownTypes(testGV, &fakeApiImpl{}, &fakeProviderConfigImpl{})

	// init cluster with objects
	fakeClient := fake.NewClientBuilder().
		WithObjects(clusterObjects...).
		WithScheme(scheme).
		WithStatusSubresource(&fakeApiImpl{}).
		Build()
	return clusters.NewTestClusterFromClient(id, fakeClient)
}

var _ API = &fakeApiImpl{}

type fakeApiImpl struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	common.Status     `json:"status,omitempty"`
}

func (f *fakeApiImpl) DeepCopyObject() runtime.Object {
	return &fakeApiImpl{
		ObjectMeta: *f.ObjectMeta.DeepCopy(),
		Status: common.Status{
			ObservedGeneration: f.ObservedGeneration,
			Phase:              f.Phase,
			Conditions:         slices.Clone(f.Conditions),
		},
	}
}

func (f *fakeApiImpl) Finalizer() string {
	return "fakeFinalizer"
}

func (f *fakeApiImpl) GetConditions() *[]metav1.Condition {
	return &f.Conditions
}

func (f *fakeApiImpl) GetStatus() any {
	return f.Status
}

func (f *fakeApiImpl) SetPhase(phase string) {
	f.Phase = phase
}
func (f *fakeApiImpl) SetObservedGeneration(g int64) {
	f.ObservedGeneration = g
}

// MockSecretWatchingReconciler satisfies both ServiceProviderReconciler and SecretWatcher.
type MockSecretWatchingReconciler struct {
	MockServiceProviderReconciler
	referencedSecrets map[string]bool
}

var _ SecretWatcher[*fakeProviderConfigImpl] = &MockSecretWatchingReconciler{}

func (m *MockSecretWatchingReconciler) IsReferencedSecret(_ context.Context, secret *corev1.Secret, _ *fakeProviderConfigImpl) bool {
	return m.referencedSecrets[secret.Name]
}

// MockConfigMapWatchingReconciler satisfies both ServiceProviderReconciler and ConfigMapWatcher.
type MockConfigMapWatchingReconciler struct {
	MockServiceProviderReconciler
	referencedConfigMaps map[string]bool
}

var _ ConfigMapWatcher[*fakeProviderConfigImpl] = &MockConfigMapWatchingReconciler{}

func (m *MockConfigMapWatchingReconciler) IsReferencedConfigMap(_ context.Context, configMap *corev1.ConfigMap, _ *fakeProviderConfigImpl) bool {
	return m.referencedConfigMaps[configMap.Name]
}

// createFakeClusterWithUnstructuredList creates a fake cluster whose client supports
// listing unstructured objects by intercepting List calls and populating the result
// from the given objects.
func createFakeClusterWithUnstructuredList(t *testing.T, id string, objs []client.Object) *clusters.Cluster {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	scheme.AddKnownTypes(testGV, &fakeApiImpl{}, &fakeProviderConfigImpl{})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if ul, ok := list.(*unstructured.UnstructuredList); ok {
					for _, obj := range objs {
						u := unstructured.Unstructured{}
						u.SetName(obj.GetName())
						u.SetNamespace(obj.GetNamespace())
						ul.Items = append(ul.Items, u)
					}
					return nil
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()
	return clusters.NewTestClusterFromClient(id, fakeClient)
}

func TestMapSecretToRequests(t *testing.T) {
	const secretName = "my-secret"
	tests := []struct {
		name           string
		secret         *corev1.Secret
		referenced     map[string]bool
		providerConfig *fakeProviderConfigImpl
		existingObjs   []client.Object
		wantRequests   int
	}{
		{
			name: "referenced secret with existing objects triggers reconciliation",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{secretName: true},
			providerConfig: &fakeProviderConfigImpl{FakePollInterval: time.Hour},
			existingObjs: []client.Object{
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-1", Namespace: testNamespaceName}},
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-2", Namespace: testNamespaceName}},
			},
			wantRequests: 2,
		},
		{
			name: "unreferenced secret does not trigger reconciliation",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "other-secret", Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{secretName: true},
			providerConfig: &fakeProviderConfigImpl{FakePollInterval: time.Hour},
			existingObjs: []client.Object{
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-1", Namespace: testNamespaceName}},
			},
			wantRequests: 0,
		},
		{
			name: "referenced secret with no existing objects returns empty",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{secretName: true},
			providerConfig: &fakeProviderConfigImpl{FakePollInterval: time.Hour},
			existingObjs:   nil,
			wantRequests:   0,
		},
		{
			name: "nil provider config does not panic",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{secretName: true},
			providerConfig: nil,
			existingObjs:   nil,
			wantRequests:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onboardingCluster := createFakeClusterWithUnstructuredList(t, "onboarding", tt.existingObjs)

			mockSW := &MockSecretWatchingReconciler{
				referencedSecrets: tt.referenced,
			}

			r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
				emptyObj:          func() *fakeApiImpl { return &fakeApiImpl{} },
				onboardingCluster: onboardingCluster,
				reconciler:        mockSW,
			}
			r.providerConfig.Store(&tt.providerConfig)

			mapFn := r.mapSecretToRequests(mockSW)
			reqs := mapFn(context.Background(), tt.secret)
			assert.Equal(t, tt.wantRequests, len(reqs))

			if tt.wantRequests > 0 {
				names := make(map[string]bool)
				for _, req := range reqs {
					names[req.Name] = true
				}
				for _, obj := range tt.existingObjs {
					assert.True(t, names[obj.GetName()], "expected request for object %s", obj.GetName())
				}
			}
		})
	}
}

func TestAPIReconciler_enqueueAllObjects(t *testing.T) {
	tests := []struct {
		name              string // description of this test case
		onboardingCluster *clusters.Cluster
		wantErrorMessage  string
		want              []reconcile.Request
	}{
		{
			name: "expect reconcile requests",
			onboardingCluster: createFakeClusterWithUnstructuredList(t, "onboarding", []client.Object{
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-1", Namespace: testNamespaceName}},
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-2", Namespace: testNamespaceName}},
			}),
			want: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "obj-1", Namespace: testNamespaceName}},
				{NamespacedName: types.NamespacedName{Name: "obj-2", Namespace: testNamespaceName}},
			},
		},
		{
			name:              "expect gvk error log without registering fake api scheme",
			onboardingCluster: clusters.NewTestClusterFromClient("onboarding", fake.NewClientBuilder().Build()),
			wantErrorMessage:  "failed to retrieve gvk",
			want:              nil,
		},
	}
	for _, tt := range tests {
		core, observedLogs := observer.New(zap.ErrorLevel)
		testContext := log.IntoContext(context.Background(), zapr.NewLogger(zap.New(core)))
		t.Run(tt.name, func(t *testing.T) {
			r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
				emptyObj:          func() *fakeApiImpl { return &fakeApiImpl{} },
				onboardingCluster: tt.onboardingCluster,
			}
			got := r.enqueueAllObjects(testContext)
			if len(got) == 0 {
				logs := observedLogs.All()
				require.Len(t, logs, 1)
				assert.Equal(t, zap.ErrorLevel, logs[0].Level)
				assert.Equal(t, tt.wantErrorMessage, logs[0].Message)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapConfigMapToRequests(t *testing.T) {
	const configMapName = "my-configmap"
	tests := []struct {
		name           string
		configMap      *corev1.ConfigMap
		referenced     map[string]bool
		providerConfig *fakeProviderConfigImpl
		existingObjs   []client.Object
		wantRequests   int
	}{
		{
			name: "referenced ConfigMap with existing objects triggers reconciliation",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{configMapName: true},
			providerConfig: &fakeProviderConfigImpl{FakePollInterval: time.Hour},
			existingObjs: []client.Object{
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-1", Namespace: testNamespaceName}},
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-2", Namespace: testNamespaceName}},
			},
			wantRequests: 2,
		},
		{
			name: "unreferenced ConfigMap does not trigger reconciliation",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "other-configmap", Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{configMapName: true},
			providerConfig: &fakeProviderConfigImpl{FakePollInterval: time.Hour},
			existingObjs: []client.Object{
				&fakeApiImpl{ObjectMeta: metav1.ObjectMeta{Name: "obj-1", Namespace: testNamespaceName}},
			},
			wantRequests: 0,
		},
		{
			name: "referenced ConfigMap with no existing objects returns empty",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{configMapName: true},
			providerConfig: &fakeProviderConfigImpl{FakePollInterval: time.Hour},
			existingObjs:   nil,
			wantRequests:   0,
		},
		{
			name: "nil provider config does not panic",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: configMapName, Namespace: testNamespaceName},
			},
			referenced:     map[string]bool{configMapName: true},
			providerConfig: nil,
			existingObjs:   nil,
			wantRequests:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onboardingCluster := createFakeClusterWithUnstructuredList(t, "onboarding", tt.existingObjs)

			mockCW := &MockConfigMapWatchingReconciler{
				referencedConfigMaps: tt.referenced,
			}

			r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
				emptyObj:          func() *fakeApiImpl { return &fakeApiImpl{} },
				onboardingCluster: onboardingCluster,
				reconciler:        mockCW,
			}
			r.providerConfig.Store(&tt.providerConfig)

			mapFn := r.mapConfigMapToRequests(mockCW)
			reqs := mapFn(context.Background(), tt.configMap)
			assert.Equal(t, tt.wantRequests, len(reqs))

			if tt.wantRequests > 0 {
				names := make(map[string]bool)
				for _, req := range reqs {
					names[req.Name] = true
				}
				for _, obj := range tt.existingObjs {
					assert.True(t, names[obj.GetName()], "expected request for object %s", obj.GetName())
				}
			}
		})
	}
}
