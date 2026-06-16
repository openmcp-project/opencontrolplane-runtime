package serviceprovider

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapWatcher can optionally be implemented by a Reconciler to trigger reconciliation
// of all API objects when a referenced ConfigMap in the provider namespace changes.
// The watch is set up on the platform cluster and filtered to the namespace configured via WithConfigMapNamespace.
type ConfigMapWatcher[PC Config] interface {
	// IsReferencedConfigMap returns true if the given ConfigMap should trigger
	// reconciliation. pc is the current provider config — it will be the
	// zero value (nil for pointer types) if not yet loaded; implementations
	// must guard against this.
	IsReferencedConfigMap(ctx context.Context, configMap *corev1.ConfigMap, pc PC) bool
}
