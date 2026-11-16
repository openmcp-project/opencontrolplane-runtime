package serviceprovider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func readyCondition(observedGeneration int64) metav1.Condition {
	return metav1.Condition{
		Type:               ServiceProviderConditionReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: observedGeneration,
		Reason:             "ReconcileSuccess",
		Message:            "ready",
		LastTransitionTime: metav1.Now(),
	}
}

func notReadyCondition(observedGeneration int64) metav1.Condition {
	return metav1.Condition{
		Type:               ServiceProviderConditionReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: observedGeneration,
		Reason:             "ReconcileError",
		Message:            "not ready",
		LastTransitionTime: metav1.Now(),
	}
}

func updatingCondition(ago time.Duration) metav1.Condition {
	return metav1.Condition{
		Type:               serviceProviderConditionUpdating,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 2,
		Reason:             "GenerationChanged",
		Message:            "Update in progress",
		LastTransitionTime: metav1.NewTime(time.Now().Add(-ago)),
	}
}

func emptyReconciler() *APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl] {
	return &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{}
}

func objWithGeneration(generation int64, createdAgo time.Duration, conditions []metav1.Condition) *fakeApiImpl {
	obj := &fakeApiImpl{}
	obj.Generation = generation
	obj.CreationTimestamp = metav1.NewTime(time.Now().Add(-createdAgo))
	obj.Conditions = conditions
	return obj
}

func TestCalcObservations(t *testing.T) {
	r := emptyReconciler()

	t.Run("create: not ready to ready, generation 1 -> create duration", func(t *testing.T) {
		const createdAgo = 5 * time.Second
		obj := objWithGeneration(1, createdAgo, []metav1.Condition{readyCondition(1)})
		oldConditions := []metav1.Condition{notReadyCondition(1)}

		create, update := r.calcObservations(obj, oldConditions)

		assert.Greater(t, create, createdAgo, "create adds computation time")
		assert.Less(t, create, createdAgo+1*time.Second, "the added computation time is expected to be very low")
		assert.Zero(t, update)
	})

	t.Run("update: not ready to ready, generation > 1 with updating condition -> update duration", func(t *testing.T) {
		const createdAgo = 60 * time.Second
		const updatingAgo = 5 * time.Second
		obj := objWithGeneration(2, createdAgo, []metav1.Condition{
			readyCondition(2),
			updatingCondition(updatingAgo),
		})
		oldConditions := []metav1.Condition{notReadyCondition(1)}

		create, update := r.calcObservations(obj, oldConditions)

		assert.Zero(t, create)
		assert.Greater(t, update, updatingAgo, "update adds computation time")
		assert.Less(t, update, updatingAgo+1*time.Second, "the added computation time is expected to be very low")
	})

	t.Run("update: not ready to ready, generation > 1 without updating condition -> no observation", func(t *testing.T) {
		obj := objWithGeneration(2, 60*time.Second, []metav1.Condition{readyCondition(2)})
		oldConditions := []metav1.Condition{notReadyCondition(1)}

		create, update := r.calcObservations(obj, oldConditions)

		assert.Zero(t, create)
		assert.Zero(t, update)
	})

	t.Run("ready to ready -> no observation", func(t *testing.T) {
		obj := objWithGeneration(1, 5*time.Second, []metav1.Condition{readyCondition(1)})
		oldConditions := []metav1.Condition{readyCondition(1)}

		create, update := r.calcObservations(obj, oldConditions)

		assert.Zero(t, create)
		assert.Zero(t, update)
	})

	t.Run("not ready to not ready -> no observation", func(t *testing.T) {
		obj := objWithGeneration(1, 5*time.Second, []metav1.Condition{notReadyCondition(1)})
		oldConditions := []metav1.Condition{notReadyCondition(1)}

		create, update := r.calcObservations(obj, oldConditions)

		assert.Zero(t, create)
		assert.Zero(t, update)
	})
}

func TestUpdateStatusAndObserve(t *testing.T) {
	req := func(name string) types.NamespacedName {
		return types.NamespacedName{Name: name, Namespace: testNamespaceName}
	}

	t.Run("no status change -> no patch, no observation", func(t *testing.T) {
		obj := &fakeApiImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectName,
				Namespace: testNamespaceName,
			},
		}
		StatusReady(obj)
		oldObj := obj.DeepCopyObject().(*fakeApiImpl)

		onboardingCluster := createFakeCluster(t, "onboarding", obj)
		r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
			onboardingCluster: onboardingCluster,
		}

		err := r.updateStatusAndObserve(context.Background(), obj, oldObj)
		require.NoError(t, err)

		// object in cluster should be unchanged
		stored := &fakeApiImpl{}
		stored.SetName(testObjectName)
		stored.SetNamespace(testNamespaceName)
		require.NoError(t, onboardingCluster.Client().Get(context.Background(), client.ObjectKeyFromObject(stored), stored))
		assert.Equal(t, StatusPhaseReady, stored.Phase)
	})

	t.Run("no status to ready -> patch is applied", func(t *testing.T) {
		obj := &fakeApiImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectName,
				Namespace: testNamespaceName,
			},
		}
		oldObj := obj.DeepCopyObject().(*fakeApiImpl)
		StatusReady(obj)

		onboardingCluster := createFakeCluster(t, "onboarding", obj)
		r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
			onboardingCluster: onboardingCluster,
		}

		err := r.updateStatusAndObserve(context.Background(), obj, oldObj)
		require.NoError(t, err)

		stored := &fakeApiImpl{}
		stored.SetName(testObjectName)
		stored.SetNamespace(testNamespaceName)
		require.NoError(t, onboardingCluster.Client().Get(context.Background(), req(testObjectName), stored))
		assert.Equal(t, StatusPhaseReady, stored.Phase)
	})

	t.Run("object not found -> no error", func(t *testing.T) {
		obj := &fakeApiImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testObjectName,
				Namespace: testNamespaceName,
			},
		}
		oldObj := obj.DeepCopyObject().(*fakeApiImpl)
		StatusReady(obj)

		// cluster has no objects
		onboardingCluster := createFakeCluster(t, "onboarding")
		r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
			onboardingCluster: onboardingCluster,
		}

		err := r.updateStatusAndObserve(context.Background(), obj, oldObj)
		require.NoError(t, err)
	})

	t.Run("generation changed -> updating condition added to newObj", func(t *testing.T) {
		obj := &fakeApiImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name:       testObjectName,
				Namespace:  testNamespaceName,
				Generation: 2,
			},
		}
		oldObj := obj.DeepCopyObject().(*fakeApiImpl)
		StatusProgressing(oldObj, reasonReconcileError, "still progressing")
		oldObj.Status.Conditions[0].ObservedGeneration = 1

		StatusProgressing(obj, reasonReconcileError, "progressing")

		onboardingCluster := createFakeCluster(t, "onboarding", obj)
		r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
			onboardingCluster: onboardingCluster,
		}

		err := r.updateStatusAndObserve(context.Background(), obj, oldObj)
		require.NoError(t, err)

		updating := meta.FindStatusCondition(obj.Conditions, serviceProviderConditionUpdating)
		assert.NotNil(t, updating, "Updating condition should have been set on generation change")
	})

	t.Run("not ready to ready after update -> Updating condition removed, update observed", func(t *testing.T) {
		const updatingAgo = 2 * time.Second
		obj := &fakeApiImpl{
			ObjectMeta: metav1.ObjectMeta{
				Name:       testObjectName,
				Namespace:  testNamespaceName,
				Generation: 2,
			},
		}
		obj.Conditions = []metav1.Condition{
			readyCondition(2),
			updatingCondition(updatingAgo),
		}

		oldObj := obj.DeepCopyObject().(*fakeApiImpl)
		// make old state not-ready so calcObservations triggers update path
		meta.SetStatusCondition(&oldObj.Conditions, notReadyCondition(1))
		meta.RemoveStatusCondition(&oldObj.Conditions, serviceProviderConditionUpdating)

		onboardingCluster := createFakeCluster(t, "onboarding", obj)
		r := &APIReconciler[*fakeApiImpl, *fakeProviderConfigImpl]{
			onboardingCluster: onboardingCluster,
		}

		err := r.updateStatusAndObserve(context.Background(), obj, oldObj)
		require.NoError(t, err)

		// Updating condition must be removed from the in-memory object
		updating := meta.FindStatusCondition(obj.Conditions, serviceProviderConditionUpdating)
		assert.Nil(t, updating, "Updating condition should be removed after update completes")
	})
}
