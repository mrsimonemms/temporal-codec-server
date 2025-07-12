# temporal-codec-server

![Version: 1.0.0](https://img.shields.io/badge/Version-1.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

Deploy Temporal Codec Server

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Node affinity |
| autoscaling.enabled | bool | `false` | Autoscaling enabled |
| autoscaling.maxReplicas | int | `100` | Maximum replicas |
| autoscaling.minReplicas | int | `1` | Minimum replicas |
| autoscaling.targetCPUUtilizationPercentage | int | `80` | When to trigger a new replica |
| config.basicPassword | string | `nil` | Optionally allow HTTP Basic to be used for /decode endpoints - also requires username |
| config.basicUsername | string | `nil` | Optionally allow HTTP Basic to be used for /decode endpoints - also requires password |
| config.corsAllowCreds | bool | `true` | Allow credentials to be sent through CORS |
| config.corsOrigins | list | `["https://cloud.temporal.io"]` | Origins allowed to use CORS |
| config.disableAuth | bool | `false` | Disable authentication |
| config.disableCors | bool | `false` | Disable CORS |
| config.disableSwagger | bool | `false` | Disable Swagger |
| config.logLevel | string | `"info"` | Log level |
| config.pause | string | `"0s"` | Pause before resolving the /decode and /encode endpoints |
| env | list | `[]` |  |
| fullnameOverride | string | `""` | String to fully override names |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.repository | string | `"ghcr.io/mrsimonemms/temporal-codec-server/golang"` | Image repositiory |
| image.tag | string | `""` | Image tag - defaults to the chart's `AppVersion` if not set |
| imagePullSecrets | list | `[]` | Docker registry secret names |
| ingress.annotations | object | `{}` | Ingress annotations |
| ingress.className | string | `"nginx"` | Ingress class name, defaulting to [ingress-nginx](https://github.com/kubernetes/ingress-nginx) |
| ingress.enabled | bool | `false` | Enable ingress |
| ingress.host | string | `"codec.temporal.local"` | Domain to use for incoming requests |
| ingress.pathType | string | `"Prefix"` | Type for the root path |
| ingress.tls.enabled | bool | `true` | Enable TLS termination for requests |
| keys.createSecret | bool | `true` | Create the keys secret |
| keys.encryptionKeys | list | `[{"id":"laqcg6jzc3kx","key":"rgQfsrQKyLGWGoYPbWOn2KfwhdRueoLU"},{"id":"3xkyy9d0a1av","key":"54APIwgWHhF0bM365vdocJvXxEQNnw88"}]` | Encryption keys to use - these are examples to show the format used and should **NOT** be used |
| keys.existingSecret | string | `""` | Use an existing secret to populate the keys |
| livenessProbe.httpGet.path | string | `"/livez"` |  |
| livenessProbe.httpGet.port | string | `"http"` |  |
| nameOverride | string | `""` | String to partially override name |
| nodeSelector | object | `{}` | Node selector |
| podAnnotations | object | `{}` | Pod [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| podLabels | object | `{}` | Pod [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |
| podSecurityContext | object | `{}` | Pod's [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context) |
| readinessProbe.httpGet.path | string | `"/livez"` |  |
| readinessProbe.httpGet.port | string | `"http"` |  |
| replicaCount | int | `1` | Number of replicas |
| resources | object | `{}` | Configure resources available |
| securityContext | object | `{}` | Container's security context |
| service.port | int | `3000` | Service's port |
| service.type | string | `"ClusterIP"` | Service's type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.automount | bool | `true` | Automatically mount a ServiceAccount's API credentials? |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| tolerations | list | `[]` | Node toleration |
| volumeMounts | list | `[]` | Additional volumeMounts on the output Deployment definition. |
| volumes | list | `[]` | Additional volumes on the output Deployment definition. |

