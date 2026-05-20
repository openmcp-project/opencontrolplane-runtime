package serviceprovider

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// SecretWatcher can optionally be implemented by a Reconciler to trigger reconciliation
// of all API objects when a referenced secret in the provider namespace changes.
// The watch is set up on the platform cluster and filtered to the namespace configured via WithSecretNamespace.
type SecretWatcher[PC ProviderConfig] interface {
	// IsReferencedSecret returns true if the given secret should trigger
	// reconciliation. pc is the current provider config — it will be the
	// zero value (nil for pointer types) if not yet loaded; implementations
	// must guard against this.
	IsReferencedSecret(ctx context.Context, secret *corev1.Secret, pc PC) bool
}
