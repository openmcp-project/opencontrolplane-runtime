# opencontrolplane-runtime

`opencontrolplane-runtime` is a shared Go library for building OpenControlPlane `ServiceProviders`, `PlatformServices` and `ClusterProvider`.

## Development

This project uses [task](https://taskfile.dev/) for code generation, validation, and tests. Run the following tasks:

1. `task generate` to update Go modules and apply formatting.
2. `task test:generate` to regenerate testdata DeepCopy files and CRD manifests.
3. `task validate` to run vet, lint, and import-format validation.
4. `task test` to set up envtest and run the test suite.

The test task configures [envtest](https://sigs.k8s.io/controller-runtime/tools/setup-envtest) automatically.

## Multicluster service providers

Use `MustBuildMulticluster` and `SetupWithMulticlusterManager` to reconcile service
objects directly in multiple onboarding clusters. The caller supplies a
multicluster-runtime manager and its cluster provider. The runtime has no KCP or
installation-specific dependency.

By default, platform access identities include the cluster name so identical
object names in different tenants cannot collide. Installations with existing,
globally unique onboarding namespaces can supply `MulticlusterAccessKey`. That
mapper must validate the cluster-to-namespace registration before returning an
identity; returning an error prevents access reconciliation for that object.

Tenant removal does not imply successful service deletion. Delete service objects
while their API is reachable, then remove the cluster registration. Unavailable
clusters are retried; cleanup after permanent cluster loss belongs to the caller.
The single-cluster builder and reconciler remain supported.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/opencontrolplane-runtime/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](https://github.com/openmcp-project/.github/blob/main/CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/neonephos/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright OpenControlPlane contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/opencontrolplane-runtime).

---

<p align="center">
  <a href="https://apeirora.eu/content/projects/">
    <img alt="BMWK-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="300"/>
  </a>
</p>

<p align="center">
  OpenControlPlane is part of <a href="https://apeirora.eu/content/projects/">ApeiroRA</a>, an EU Important Project of Common European Interest (IPCEI-CIS).
</p>

<p align="center">
  Copyright Linux Foundation Europe. For web site terms of use, trademark policy and other project policies please see <a href="https://linuxfoundation.eu/en/policies">https://linuxfoundation.eu/en/policies</a>.
</p>
