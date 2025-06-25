<!--
(NOTE: Do not edit README.md directly. It is a generated file!)
(      To make changes, please modify README.md.gotmpl and run `helm-docs`)
-->

# alloy-operator

![Version: 0.3.2](https://img.shields.io/badge/Version-0.3.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.1.1](https://img.shields.io/badge/AppVersion-1.1.1-informational?style=flat-square)

A Helm chart the Alloy Operator, a project to innovate on creating instances of Grafana Alloy.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| petewall | <pete.wall@grafana.com> |  |

<!-- markdownlint-disable no-bare-urls -->
<!-- markdownlint-disable list-marker-space -->
## Source Code

* <https://github.com/grafana/alloy-operator>
<!-- markdownlint-enable list-marker-space -->

## Requirements

| Repository | Name | Version |
|------------|------|---------|
|  | podlogs-crd | 0.0.0 |
| https://grafana.github.io/helm-charts | alloy-crd | 1.0.0 |
<!-- markdownlint-enable no-bare-urls -->

## Values

### Pod Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Set the affinity for the Alloy Operator pods. |
| nodeSelector | object | `{"kubernetes.io/os":"linux"}` | Set the node selector for the Alloy Operator pods. |
| podAnnotations | object | `{}` | Additional annotations to add to the Alloy Operator pods. |
| podLabels | object | `{}` | Additional labels to add to the Alloy Operator pods. |
| podSecurityContext | object | `{}` | Set the security context for the Alloy Operator pods. Example: podSecurityContext: { fsGroup: 2000 } |
| priorityClassName | string | `""` | Sets the priority class name for the Alloy Operator pods. |
| tolerations | list | `[]` | Set the tolerations for the Alloy Operator pods. |

### CRDs

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| crds.deployAlloyCRD | bool | `true` | Should this chart deploy the Alloy CRD? |
| crds.deployPodLogsCRD | bool | `false` | Should this chart deploy the PodLogs CRD? |

### Image Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| global.image.pullSecrets | list | `[]` | Global image pull secrets. |
| global.image.registry | string | `""` | Global image registry override. |
| image.pullPolicy | string | `"IfNotPresent"` | The pull policy for images. |
| image.pullSecrets | list | `[]` | Optional set of image pull secrets. |
| image.registry | string | `"ghcr.io"` | Alloy Operator image registry |
| image.repository | string | `"grafana/alloy-operator"` | Alloy Operator image repository |
| image.tag | string | `""` | Alloy Operator image tag. When empty, the Chart's appVersion is used. |

### Probes

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| livenessProbe | object | `{"initialDelaySeconds":15,"periodSeconds":20}` | Liveness probe settings |
| readinessProbe | object | `{"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe settings |

### Alloy Management Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| namespaces | list | `[]` | Restrict the Alloy Operator to only manage Alloy instances in the given list of namespaces. |
| ownNamespaceOnly | bool | `false` | Restrict the Alloy Operator to its own namespace only. Overrides the `namespaces` setting. |

### RBAC Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| rbac.create | bool | `true` | Whether to create the necessary RBAC resources for the Alloy Operator. |
| rbac.createClusterRoles | bool | `true` | Create ClusterRoles for the Alloy Operator. If set to false, only Roles and RoleBindings will be created. This setting requires the use of `namespaces` or `ownNamespaceOnly` to be set. |

### Deployment Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| replicaCount | int | `1` | How many replicas to use for the Alloy Operator Deployment. |

### Resources

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| resources.limits | object | `{}` | Set the resource limits for the Alloy Operator pods. |
| resources.requests | object | `{}` | Set the resource requests for the Alloy Operator pods. |

### Container Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| securityContext | object | `{"runAsNonRoot":true}` | Set the security context for the operator container. |

### Service

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| service.health.port | int | `8081` | The port number for the health probes. |
| service.metrics.port | int | `8082` | The port number for the metrics service. |
| service.type | string | `"ClusterIP"` | The type of service to create for the operator. |

### Service Account Settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.automount | bool | `true` | Whether the Alloy Operator pod should automatically mount the service account token. |
| serviceAccount.create | bool | `true` | Whether to create a service account for the Alloy Operator deployment. |
| serviceAccount.labels | object | `{}` | Additional labels to add to the service account |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |

### Other Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| fullnameOverride | string | `""` | Overrides the chart's computed fullname. Used to change the full prefix of resource names. |
| nameOverride | string | `""` | Overrides the chart's name. Used to change the infix in the resource names. |
