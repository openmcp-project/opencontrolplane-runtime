/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serviceprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openmcp-project/controller-utils/pkg/clusters"
	"github.com/openmcp-project/opencontrolplane-runtime/pkg/serviceprovider/clusteraccess"
	apiv1alpha1 "github.com/openmcp-project/opencontrolplane-runtime/testdata/api/v1alpha1"
	configv1alpha1 "github.com/openmcp-project/opencontrolplane-runtime/testdata/config/v1alpha1"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/openmcp-project/openmcp-operator/api/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx              context.Context
	cancel           context.CancelFunc
	platformEnv      *envtest.Environment
	onboardingEnv    *envtest.Environment
	platformCfg      *rest.Config
	onboardingCfg    *rest.Config
	platformClient   client.Client
	platformCluster  *clusters.Cluster
	onboardingClient client.Client
	onboardingScheme *runtime.Scheme
	platformScheme   *runtime.Scheme
	reconciler       *MockFooReconciler
)

func TestReconcilers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Reconciler Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	onboardingScheme = runtime.NewScheme()
	err = apiv1alpha1.AddToScheme(onboardingScheme)
	Expect(err).NotTo(HaveOccurred())

	platformScheme = runtime.NewScheme()
	err = configv1alpha1.AddToScheme(platformScheme)
	Expect(err).NotTo(HaveOccurred())

	// create platform environment
	createPlatformEnv()
	createOnboardingEnv()

	mgr, err := ctrl.NewManager(onboardingCfg, ctrl.Options{
		Scheme: onboardingScheme,
	})
	Expect(err).NotTo(HaveOccurred())

	err = fooReconciler().SetupWithManager(mgr, "foo")
	Expect(err).NotTo(HaveOccurred())
	mgr.Add(platformCluster.Cluster())

	go func() {
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
	Expect(mgr.GetCache().WaitForCacheSync(ctx)).To(BeTrue())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := platformEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = onboardingEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

func createPlatformEnv() {
	By("bootstrapping platform test environment")
	platformEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "testdata", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		platformEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// platformCfg is defined in this file globally.
	var err error
	platformCfg, err = platformEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(platformCfg).NotTo(BeNil())

	platformClient, err = client.New(platformCfg, client.Options{Scheme: platformScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(platformClient).NotTo(BeNil())
}

func createOnboardingEnv() {
	By("bootstrapping onboarding test environment")
	onboardingEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "testdata", "api", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		onboardingEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}
	// onboardingCfg is defined in this file globally.
	var err error
	onboardingCfg, err = onboardingEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(onboardingCfg).NotTo(BeNil())

	onboardingClient, err = client.New(onboardingCfg, client.Options{Scheme: onboardingScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(onboardingClient).NotTo(BeNil())
}

func fooReconciler() *APIReconciler[*apiv1alpha1.Foo, *configv1alpha1.ProviderConfig] {
	onboardingCluster := clusters.New("onboarding").WithRESTConfig(onboardingCfg)
	if err := onboardingCluster.InitializeClient(onboardingScheme); err != nil {
		panic(err)
	}
	platformCluster = clusters.New("platform").WithRESTConfig(platformCfg)
	if err := platformCluster.InitializeClient(platformScheme); err != nil {
		panic(err)
	}
	reconciler = &MockFooReconciler{config: configv1alpha1.ProviderConfig{
		Spec: configv1alpha1.ProviderConfigSpec{
			PollInterval: &metav1.Duration{Duration: time.Second},
		},
	}}
	builder := NewAPIReconcilerBuilder[*apiv1alpha1.Foo, *configv1alpha1.ProviderConfig]().
		EmptyObjectProvider(func() *apiv1alpha1.Foo { return &apiv1alpha1.Foo{} }).
		EmptyConfigProvider(func() *configv1alpha1.ProviderConfig { return &configv1alpha1.ProviderConfig{} }).
		OnboardingCluster(onboardingCluster).
		PlatformCluster(platformCluster).
		AdvancedClusterAccessReconciler(FakeAdvancedClusterAccessProvider{
			clusters: map[string]*clusters.Cluster{
				clusteraccess.MCPClusterID: {},
			},
			accessRequests: map[string]*clustersv1alpha1.AccessRequest{
				clusteraccess.MCPClusterID: {
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
			},
		}).
		Reconciler(reconciler).
		WorkloadCluster(false)
	return builder.MustBuild()
}

var _ Reconciler[*apiv1alpha1.Foo, *configv1alpha1.ProviderConfig] = &MockFooReconciler{}

type MockFooReconciler struct {
	createOrUpdateCalled bool
	deleteCalled         bool
	config               configv1alpha1.ProviderConfig
}

// CreateOrUpdate implements [Reconciler].
func (m *MockFooReconciler) CreateOrUpdate(ctx context.Context, obj *apiv1alpha1.Foo, config *configv1alpha1.ProviderConfig, clusters clusteraccess.ClusterContext) (ctrl.Result, error) {
	m.createOrUpdateCalled = true
	m.config = *config
	StatusReady(obj)
	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// Delete implements [Reconciler].
func (m *MockFooReconciler) Delete(ctx context.Context, obj *apiv1alpha1.Foo, config *configv1alpha1.ProviderConfig, clusters clusteraccess.ClusterContext) (ctrl.Result, error) {
	m.deleteCalled = true
	m.config = *config
	StatusTerminating(obj)
	return ctrl.Result{}, nil
}
