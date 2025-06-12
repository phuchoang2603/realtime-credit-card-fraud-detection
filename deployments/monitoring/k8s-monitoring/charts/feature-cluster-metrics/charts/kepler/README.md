# Kepler
Kepler (Kubernetes-based Efficient Power Level Exporter) uses eBPF to probe energy related system stats and exports as Prometheus metrics

## Parameters

| Name                         | Description                            | Value        |
| ---------------------------- | -------------------------------------- | ------------ |
| `nameOverride`               | overrides the name of the chart        | `""`         |
| `fullnameOverride`           | replaces the generated name            | `""`         |
| `image.repository`           | repository to pull the image from      | `"quay.io/sustainable_computing_io/kepler"`            |
| `image.tag`                  | image tag defaults to chart appVersion | `""`         |
| `image.pullPolicy`           | image pull policy                      | `Always`     |
| `imagePullSecrets`           | Secret name for pulling images from private repository                | `[]` |
| `podAnnotations`             | Additional pod annotations             | `{}`         |
| `podSecurityContext`         | privileges and access control settings for a Pod | `{}` |
| `securityContext.privileged` | privileges and access control settings | `true`       |
| `nodeSelector`               | node selection constraint              | `{}` |
| `tolerations[].effect`       | toleration effect                      | `NoSchedule` |
| `tolerations[].key`          | toleration key                         | `node-role.kubernetes.io/master` |
| `affinity`                   | affinity rules                         | `{}` |

## Resources

| Name                         | Description                           | Value        |
| ---------------------------- | ------------------------------------- | ------------ |
| `resources.requests.cpu`     | cpu request                           | `100m`       |
| `resources.requests.memory`  | memory request                        | `200Mi`      |
| `resources.limits.cpu`       | cpu limit                             | `100m`       |
| `resources.limits.memory`    | memory limit                          | `200Mi`      |

## Environment Variables

| Name                                     | Description                    | Value    |
| ---------------------------------------- | ------------------------------ | -------- |
| `extraEnvVars.KEPLER_LOG_LEVEL`          | the kepler log level           | `"1"`    |
| `extraEnvVars.ENABLE_GPU`                | enable GPU                     | `"true"` |
| `extraEnvVars.ENABLE_EBPF_CGROUPID`      | enable EBPF CGROUPID           | `"true"` |
| `extraEnvVars.EXPOSE_IRQ_COUNTER_METRICS`| expose IRQ Counter metrics     | `"true"` |
| `extraEnvVars.EXPOSE_KUBELET_METRICS`    | expose kubelet metrics         | `"true"` |
| `extraEnvVars.ENABLE_PROCESS_METRICS`    | enable process metrics         | `"true"` |
| `extraEnvVars.CPU_ARCH_OVERRIDE`         | override CPU architechure      | `""`     |
| `extraEnvVars.CGROUP_METRICS`            | specify CGROUP Metrics         | `"*"`    |

## Service

| Name                         | Description                            | Value        |
| ---------------------------- | -------------------------------------- | ------------ |
| `service.annotations`        | annotations for the service            | `{}` |
| `service.type`               | the service type                       | `ClusterIP` |
| `service.port`               | the service port                       | `9102` |

## Service Account

| Name                         | Description                            | Value        |
| ---------------------------- | -------------------------------------- | ------------ |
| `serviceAccount.create`      | whether the service account is created | `false` |
| `serviceAccount.annotations` | annotations for the service account    | `{}` |
| `serviceAccount.name`        | name override                          | `""` |

## Service Monitor

| Name                          | Description                            | Value       |
| ----------------------------- | -------------------------------------- | ----------- |
| `serviceMonitor.enabled`      | whether the service monitor is enabled | `false`     |
| `serviceMonitor.namespace`    | which namespace to put it in           | `""`        |
| `serviceMonitor.interval`     | the scrape interval                    | `30s`       |
| `serviceMonitor.scrapeTimeout`| the scrape timeout                     | `5s`        |
| `serviceMonitor.labels`       | labels for the service monitor         | `{} `       |

## Redfish BMC and IPMI

| Name                  | Description                            | Value     |
| --------------------- | -------------------------------------- | --------- |
| `redfish.name`        | redfish secret name                    | `redfish` |
| `redfish.enabled`     | whether the redfisch secret is enabled | `false`   |
| `redfish.annotations` | annotations for redfish secret         | `{}`      |
| `redfish.fileContent` | redfish credentials                    | ``        |
| `redfish.labels`      | labels for redfish secret              | `{}`      |

## Model Server & Estimator Sidecar

| Name                              | Description                                                                               | Value     |
| --------------------------------- | ----------------------------------------------------------------------------------------- | --------- |
| `modelServer.enabled`             | whether model-server and estimator sidecar should be deployed                             | `false`   |
| `modelServer.modelConfig`         | [modelConfig](https://sustainable-computing.io/kepler_model_server/get_started/) contents | `NODE_COMPONENTS_ESTIMATOR=true` |
| `modelServer.nameOverride`        | overrides the name-suffix of the model-server deployment and service                      | `""` |
| `modelServer.fullnameOverride`    | replaces the name of the model-server deployment and service                              | `""` |
| `modelServer.replicas`            | replicas of the model-server deployment                                                   | `""` |
| `modelServer.image.repository`    | repository to pull the model-server image from                                            | `"quay.io/sustainable_computing_io/kepler_model_server"` |
| `modelServer.image.tag`           | image tag for the model-server                                                            | `"v0.7.12"` |
| `modelServer.image.pullPolicy`    | image pull policy for the model-server image                                              | `Always`    |
| `modelServer.imagePullSecrets`    | Secret name for pulling model-server images from private repository                       | `[]`        |
| `modelServer.podAnnotations`      | Additional pod annotations for the model-server pods                                      | `{}`        |
| `modelServer.securityContext`     | privileges and access control settings for the model-server container                     | `{}`        |
| `modelServer.podSecurityContext`  | privileges and access control settings for model-server pods                              | `{}`        |
| `modelServer.resources`           | resource limits and requests for the model-server                                         | `{}`        |
| `modelServer.sidecarResources`    | resource limits and requests for the estimator sidecar                                    | `{}`        |
| `modelServer.service.annotations` | annotations for the model-server service                                                  | `{}`        |
| `modelServer.service.type`        | the model-server service type                                                             | `ClusterIP` |
| `modelServer.service.port`        | the model-server service port                                                             | `8100`      |
