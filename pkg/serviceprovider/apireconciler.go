package serviceprovider

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/openmcp-project/controller-utils/pkg/clusters"
	controllerutil2 "github.com/openmcp-project/controller-utils/pkg/controller"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	apiconst "github.com/openmcp-project/openmcp-operator/api/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// API represents the end-user facing onboarding API type
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

// APIReconciler implements a generic reconcile loop to separate platform
// and service provider developer space.
type APIReconciler[T API, PC ProviderConfig] struct {
	// platformCluster represents the platform cluster of the v2 architecture
	platformCluster *clusters.Cluster
	// onboardingCluster represents the onboarding cluster of the v2 architecture
	onboardingCluster *clusters.Cluster
	// clusterAccessReconciler reconciles access to MCP and workload clusters
	clusterAccessReconciler ClusterAccessProvider
	// reconciler reconciles the end-user facing onboarding API of a service provider
	reconciler Reconciler[T, PC]
	// providerConfig represents the platform operator facing platform API of a service provider
	providerConfig atomic.Pointer[PC]
	// withWorkloadCluster defines whether a service provider requires access to a workload cluster
	withWorkloadCluster bool
	// secretNamespace is the namespace to watch secrets in on the platform cluster. Used only if the ServiceProviderReconciler also implements SecretWatcher.
	secretNamespace string
	// emptyObj creates an empty object of the api type
	emptyObj func() T
}

// NewAPIReconciler creates a reconciler instance for the given types.
func NewAPIReconciler[T API, PC ProviderConfig](emptyObj func() T) *APIReconciler[T, PC] {
	return &APIReconciler[T, PC]{
		emptyObj: emptyObj,
	}
}

// WithPlatformCluster set the platform cluster.
func (r *APIReconciler[T, PC]) WithPlatformCluster(c *clusters.Cluster) *APIReconciler[T, PC] {
	r.platformCluster = c
	return r
}

// WithOnboardingCluster set the onboarding cluster.
func (r *APIReconciler[T, PC]) WithOnboardingCluster(c *clusters.Cluster) *APIReconciler[T, PC] {
	r.onboardingCluster = c
	return r
}

// WithClusterAccessReconciler sets the cluster access reconciler.
func (r *APIReconciler[T, PC]) WithClusterAccessReconciler(car ClusterAccessProvider) *APIReconciler[T, PC] {
	r.clusterAccessReconciler = car
	return r
}

// WithServiceProviderReconciler sets the service provider reconciler.
func (r *APIReconciler[T, PC]) WithServiceProviderReconciler(dsr Reconciler[T, PC]) *APIReconciler[T, PC] {
	r.reconciler = dsr
	return r
}

// WithWorkloadCluster sets if the service provider reconciler requests a workload cluster
func (r *APIReconciler[T, PC]) WithWorkloadCluster(b bool) *APIReconciler[T, PC] {
	r.withWorkloadCluster = b
	return r
}

// WithSecretNamespace enables secret watching in the given namespace on the platform cluster. Only used if the ServiceProviderReconciler also implements SecretWatcher.
func (r *APIReconciler[T, PC]) WithSecretNamespace(ns string) *APIReconciler[T, PC] {
	r.secretNamespace = ns
	return r
}

// WithProviderConfig sets if the service provider config.
func (r *APIReconciler[T, PC]) WithProviderConfig(config PC) *APIReconciler[T, PC] {
	r.providerConfig.Store(&config)
	return r
}

// Reconcile orchestrates platform and DomainServiceReconciler logic to reconcile APIObjects
func (r *APIReconciler[T, PC]) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reconcileErr error) {
	l := logf.FromContext(ctx)
	// common reconciler logic including get obj, providerconfig, mcp/workload access
	obj := r.emptyObj()
	if err := r.onboardingCluster.Client().Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Skip reconciliation if annotation is set
	if obj.GetAnnotations()[apiconst.OperationAnnotation] == apiconst.OperationAnnotationValueIgnore {
		l.Info("Skipping resource due to ignore operation annotation")
		return ctrl.Result{}, nil
	}
	oldObj := obj.DeepCopyObject().(T)
	// always try to update the obj status
	defer func() {
		if err := r.updateStatus(ctx, obj, oldObj); err != nil {
			l.Error(err, "status update failed")
			reconcileErr = errors.Join(reconcileErr, err)
		}
	}()
	providerConfig := r.providerConfig.Load()
	if providerConfig == nil {
		StatusProgressing(obj, reasonReconcileError, "No ProviderConfig found")
		return ctrl.Result{}, errors.New("provider config missing")
	}
	providerConfigCopy := (*providerConfig).DeepCopyObject().(PC)
	// core crud
	deleted := !obj.GetDeletionTimestamp().IsZero()
	var res ctrl.Result
	var err error
	if deleted {
		res, err = r.delete(ctx, obj, providerConfigCopy)
	} else {
		res, err = r.createOrUpdate(ctx, obj, providerConfigCopy)
	}
	// return based on result/err
	if err != nil {
		l.Error(err, "reconcile failed")
		return ctrl.Result{}, err
	}
	if res.RequeueAfter > 0 {
		return res, nil
	}
	// fallback to poll interval to prevent 'managed service' drift
	return ctrl.Result{
		RequeueAfter: providerConfigCopy.PollInterval(),
	}, nil
}

func (r *APIReconciler[T, PC]) updateStatus(ctx context.Context, newObj T, oldObj T) error {
	if equality.Semantic.DeepEqual(oldObj.GetStatus(), newObj.GetStatus()) {
		return nil
	}
	err := r.onboardingCluster.Client().Status().Patch(ctx, newObj, client.MergeFrom(oldObj))
	// can't update status if object doesn't exist
	return client.IgnoreNotFound(err)
}

// delete eventually invokes the domain delete logic of a service provider and is the place to implement
// common logic that should be abstracted away from a service provider developer like handling cluster access.
func (r *APIReconciler[T, PC]) delete(ctx context.Context, obj T, pc PC) (ctrl.Result, error) {
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(obj)}
	accessRequestsInDeletion, err := r.areAccessRequestsInDeletion(ctx, req)
	if err != nil {
		StatusProgressing(obj, reasonReconcileError, "failed to check access requests in deletion")
		return reconcile.Result{}, err
	}
	if !accessRequestsInDeletion {
		clusters, res, err := r.clusters(ctx, req)
		if err != nil {
			terminatingWithReason(obj, reasonReconcileError, "cluster cleanup error")
			return ctrl.Result{}, err
		}
		if res.RequeueAfter > 0 {
			terminatingWithReason(obj, "Reconciling", "cluster cleanup")
			return res, nil
		}
		res, err = r.reconciler.Delete(ctx, obj, pc, clusters)
		if err != nil {
			return ctrl.Result{}, err
		}
		if res.RequeueAfter > 0 {
			return res, nil
		}
	}
	// remove cluster access
	res, err := r.clusterAccessReconciler.ReconcileDelete(ctx, req)
	if err != nil {
		terminatingWithReason(obj, reasonReconcileError, "failed cluster access reconcile delete")
		return ctrl.Result{}, err
	}
	// make sure to not drop the object before cleanup has been done
	if res.RequeueAfter > 0 {
		return res, nil
	}
	// remove finalizer
	controllerutil.RemoveFinalizer(obj, obj.Finalizer())
	if err := r.onboardingCluster.Client().Update(ctx, obj); err != nil {
		terminatingWithReason(obj, reasonReconcileError, "failed to remove finalizer")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// createOrUpdate eventually invokes the domain createOrUpdate logic of a service provider and is the place to implement
// common logic that should be abstracted away from a service provider developer like handling cluster access.
func (r *APIReconciler[T, PC]) createOrUpdate(ctx context.Context, obj T, pc PC) (ctrl.Result, error) {
	if _, err := controllerutil.CreateOrUpdate(ctx, r.onboardingCluster.Client(), obj, func() error {
		controllerutil.AddFinalizer(obj, obj.Finalizer())
		return nil
	}); err != nil {
		StatusProgressing(obj, reasonReconcileError, "failed to add finalizer")
		return ctrl.Result{}, err
	}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(obj)}
	clusters, res, err := r.clusters(ctx, req)
	if err != nil {
		StatusProgressing(obj, reasonReconcileError, "cluster setup error")
		return ctrl.Result{}, err
	}
	if res.RequeueAfter > 0 {
		return res, nil
	}
	return r.reconciler.CreateOrUpdate(ctx, obj, pc, clusters)
}

// areAccessRequestsInDeletion determines if the access requests for a reconcile request are in deletion.
// It returns true if any access requests (mcp, workload) is deleted or has a deletion timestamp.
// It is used to prevent renewing cluster access when deleting an ServiceProviderAPI object.
func (r *APIReconciler[T, PC]) areAccessRequestsInDeletion(ctx context.Context, req ctrl.Request) (bool, error) {
	accessRequest, err := r.clusterAccessReconciler.MCPAccessRequest(ctx, req)
	if apierrors.IsNotFound(err) || (accessRequest != nil && accessRequest.DeletionTimestamp != nil) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if r.withWorkloadCluster {
		accessRequest, err = r.clusterAccessReconciler.WorkloadAccessRequest(ctx, req)
		if apierrors.IsNotFound(err) || (accessRequest != nil && accessRequest.DeletionTimestamp != nil) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

// clusters returns any request scoped cluster that a servicer provider developer might want to access in order
// to delivery its service.
func (r *APIReconciler[T, PC]) clusters(ctx context.Context, req ctrl.Request) (ClusterContext, ctrl.Result, error) {
	clusters := ClusterContext{}
	res, err := r.clusterAccessReconciler.Reconcile(ctx, req)
	if err != nil {
		return clusters, ctrl.Result{}, err
	}
	if res.RequeueAfter > 0 {
		return clusters, res, nil
	}
	mcpCluster, err := r.clusterAccessReconciler.MCPCluster(ctx, req)
	if err != nil {
		return clusters, ctrl.Result{}, err
	}
	if mcpCluster == nil {
		return clusters, res, errors.New("mcp access missing")
	}
	clusters.MCPCluster = mcpCluster
	ar, err := r.clusterAccessReconciler.MCPAccessRequest(ctx, req)
	if err != nil {
		return clusters, ctrl.Result{}, err
	}
	clusters.MCPAccessSecretKey = retrieveSecretKey(ar)
	if r.withWorkloadCluster {
		workloadCluster, err := r.clusterAccessReconciler.WorkloadCluster(ctx, req)
		if err != nil {
			return clusters, ctrl.Result{}, err
		}
		if workloadCluster == nil {
			return clusters, res, errors.New("workload cluster access missing")
		}
		clusters.WorkloadCluster = workloadCluster
		ar, err := r.clusterAccessReconciler.WorkloadAccessRequest(ctx, req)
		if err != nil {
			return clusters, ctrl.Result{}, err
		}
		clusters.WorkloadAccessSecretKey = retrieveSecretKey(ar)
	}
	return clusters, res, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *APIReconciler[T, PC]) SetupWithManager(mgr ctrl.Manager, name string, providerConfigUpdates chan event.GenericEvent) error {
	controller := ctrl.NewControllerManagedBy(mgr).
		For(r.emptyObj()).
		// sets up reconciles whenever provider config controller sends update events
		WatchesRawSource(
			source.Channel(
				providerConfigUpdates,
				handler.EnqueueRequestsFromMapFunc(
					func(ctx context.Context, obj client.Object) []reconcile.Request {
						// update cached provider config
						if obj != nil {
							c := obj.DeepCopyObject().(PC)
							r.providerConfig.Store(&c)
						} else {
							r.providerConfig.Store(nil)
						}
						// reconcile all existing objects
						return r.enqueueAllObjects(ctx)
					},
				)),
		)

	// Optional: watch secrets on the platform cluster if the reconciler implements SecretWatcher
	if sw, ok := r.reconciler.(SecretWatcher[PC]); ok && r.secretNamespace != "" {
		controller = controller.WatchesRawSource(
			source.Kind(
				r.platformCluster.Cluster().GetCache(),
				&corev1.Secret{},
				handler.TypedEnqueueRequestsFromMapFunc(r.mapSecretToRequests(sw)),
				controllerutil2.ToTypedPredicate[*corev1.Secret](
					predicate.NewPredicateFuncs(func(obj client.Object) bool {
						return obj.GetNamespace() == r.secretNamespace
					}),
				),
			),
		)
	}

	return controller.Named(name).Complete(r)
}

// mapSecretToRequests returns a typed map function that checks whether a changed secret
// is referenced by the service provider and, if so, enqueues all ServiceProviderAPI objects.
func (r *APIReconciler[T, PC]) mapSecretToRequests(sw SecretWatcher[PC]) func(ctx context.Context, secret *corev1.Secret) []reconcile.Request {
	return func(ctx context.Context, secret *corev1.Secret) []reconcile.Request {
		var pcVal PC
		if pc := r.providerConfig.Load(); pc != nil {
			pcVal = *pc
		}
		if !sw.IsReferencedSecret(ctx, secret, pcVal) {
			return nil
		}
		return r.enqueueAllObjects(ctx)
	}
}

// enqueueAllObjects lists all ServiceProviderAPI objects and returns a reconcile request for each.
func (r *APIReconciler[T, PC]) enqueueAllObjects(ctx context.Context) []reconcile.Request {
	var list unstructured.UnstructuredList
	gvk, err := apiutil.GVKForObject(r.emptyObj(), r.onboardingCluster.Scheme())
	if err != nil {
		logf.FromContext(ctx).Error(err, "failed to retrieve gvk")
		return nil
	}
	list.SetGroupVersionKind(gvk)
	if err := r.onboardingCluster.Client().List(ctx, &list); err != nil {
		logf.FromContext(ctx).Error(err, "failed to list objects")
		return nil
	}
	reqs := make([]reconcile.Request, len(list.Items))
	for i := range list.Items {
		reqs[i] = reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(&list.Items[i]),
		}
	}
	return reqs
}

func retrieveSecretKey(ar *clustersv1alpha1.AccessRequest) client.ObjectKey {
	if ar.Status.SecretRef == nil {
		return client.ObjectKey{}
	}
	return client.ObjectKey{
		Namespace: ar.Namespace,
		Name:      ar.Status.SecretRef.Name,
	}
}
