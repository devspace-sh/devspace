---
title: Component specification
---

## component
```yaml
component:                          # struct   | Options for deploying a DevSpace component
  containers: ...                   # struct   | Relative path
  replicas: 1                       # int      | Number of replicas (Default: 1)
  autoScaling: ...                  # struct   | AutoScaling configuration
  rollingUpdate: ...                # struct   | RollingUpdate configuration
  volumes: ...                      # struct   | Component volumes
  service: ...                      # struct   | Component service
  serviceName: my-service           # string   | Service name for headless service (for StatefulSets)
  podManagementPolicy: OrderedReady # enum     | "OrderedReady" or "Parallel" (for StatefulSets)
  pullSecrets: ...                  # string[] | Array of PullSecret names
```
[Learn more about configuring component deployments.](/docs/deployment/components/what-are-components)

## component.containers
```yaml
containers:                         # struct   | Options for deploying a DevSpace component
- name: my-container                # string   | Container name (optional)
  image: dscr.io/username/image     # string   | Image name (optionally with registry URL)
  command:                          # string[] | ENTRYPOINT override
  - sleep
  args:                             # string[] | ARGS override
  - 99999
  env:                              # map[string]string | Kubernetes env definition for containers
  - name: MY_ENV_VAR
    value: "my-value"
  volumeMounts: ...                 # struct   | VolumeMount Configuration
  resources: ...                    # struct   | Kubernestes resource limits and requests
  livenessProbe: ...                # struct   | Kubernestes livenessProbe
  redinessProbe: ...                # struct   | Kubernestes redinessProbe
```

## component.containers[*].volumeMounts
```yaml
volumeMounts: 
  containerPath: /my/path           # string   | Mount path within the container
  volume:                           # struct   | Volume to mount
    name: my-volume                 # string   | Name of the volume to be mounted
    subPath: /in/my/volume          # string   | Path inside to volume to be mounted to the containerPath
    readOnly: false                 # bool     | Mount volume as read-only (Default: false)
```

## component.autoScaling
```yaml
autoScaling: 	                    # struct   | Auto-Scaling configuration
  horizontal:                       # struct   | Configuration for horizontal auto-scaling
    maxReplicas: 5                  # int      | Max replicas to deploy
    averageCPU: 800m                # string   | Target value for CPU usage
    averageMemory: 1Gi              # string   | Target value for memory (RAM) usage
```

## component.rollingUpdate
```yaml
rollingUpdate: 	                    # struct   | Rolling-Update configuration
  enabled: false                    # bool     | Enable/Disable rolling update (Default: disabled)
  maxSurge: "25%"                   # string   | Max number of pods to be created above the pod replica limit
  maxUnavailable: "50%"             # string   | Max number of pods unavailable during update process
  partition: 1                      # int      | For partitioned updates of StatefulSets
```

## component.volumes
```yaml
volumes: 	                        # struct   | Array of volumes to be created
- name: my-volume                   # string   | Volume name
  size: 10Gi                        # string   | Size of the volume in Gi (Gigabytes)
  configMap: ...                    # struct   | Kubernetes ConfigMapVolumeSource
  secret: ...                       # struct   | Kubernetes SecretVolumeSource
```

## component.service
```yaml
service: 	                        # struct   | Component service configuration
  name: my-service                  # string   | Name of the service
  type: NodePort                    # string   | Type of the service (default: NodePort)
  ports:                            # array    | Array of service ports
  - port: 80                        # int      | Port exposed by the service
    containerPort: 3000             # int      | Port of the container/pod to redirect traffic to
    protocol: tcp                   # string   | Traffic protocol (tcp, udp)
```
