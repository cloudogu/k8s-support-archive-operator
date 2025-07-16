# Usage

To create a support archive, apply the [crd lib](https://github.com/cloudogu/k8s-support-archive-lib), [operator](https://github.com/cloudogu/k8s-support-archive-operator) and a support archive custom resource in the cluster.
See [crd lib](https://github.com/cloudogu/k8s-support-archive-lib/blob/develop/k8s/helm-crd/templates/k8s.cloudogu.com_supportarchives.yaml) for the custom resource format.

The deployment of the operator contains a nginx sidecar container with a shared volume to expose created support archives.

## Internal processes

### Finalizer

The triggered reconciler first checks the finalizer existence and adds one if no one is defined.
With the finalizer, the operator can later delete support archives when the custom resource will be deleted.

### Reconciler

In general, the reconciler will always try to requeue the custom resource to avoid blocking.
This happens, e.g. after adding the finalizer or executing one single collector to fetch data.

### State

When creating the support archive, the operator always checks the metadata from the actual state of the archive first.
It is defined in a `JSON` File for each archive in `/data/work/<namespace>/<name>` and is not accessible from the nginx sidecar.
The metadata itself contains the names of the steps which fetched data for the archives (Collectors).

The operator persists the state (the resulting archive) as a `ZIP` under following path `/data/supportarchives/namespace/name`.
To avoid memory exhaustion, the state component should provide a `io.Writer` to the collectors to stream data to the filesystem.

### Collectors

Collectors are responsible to fetch individual data sections for the archive, e.g. logs, kubernetes resources, health.
They should use the exposed writer from the state component and also care about memory exhaustion.

A list of collectors defines the completeness of a support archives.