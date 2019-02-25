---
title: Full Reference
---

## version
```yaml
version: v1alpha2                   # string   | Version of the config
```

<details>
<summary>
### List of supported versions
</summary>
- v1alpha2 ***latest***
- v1alpha1
</details>

---
## images
```yaml
images:                             # map[string]struct | Images to be built and pushed
  image1:                           # string   | Name of the image
    image: dscr.io/username/image   # string   | Image repository and name 
    tag: v0.0.1                     # string   | Image tag
    createPullSecret: true          # bool     | Create a pull secret containing your Docker credentials (Default: true)
    insecure: false                 # bool     | Allow push/pull to/from insecure registries (Default: false)
    skipPush: false                 # bool     | Skip pushing image to registry, recommended for minikube (Default: false)
    build: ...                      # struct   | Build options for this image
  image2: ...
```
[Learn more about building images with DevSpace.](../images/workflow)

### images[*].build
```yaml
build:                              # struct   | Build configuration for an image
  disabled: false                   # bool     | Disable image building (Default: false)
  dockerfilePath: ./Dockerfile      # string   | Relative path to the Dockerfile used for building (Default: ./Dockerfile)
  contextPath: ./                   # string   | Relative path to the context used for building (Default: ./)
  kaniko: ...                       # struct   | Build image with kaniko and set options for kaniko
  docker: ...                       # struct   | Build image with docker and set options for docker
  options: ...                      # struct   | Set build options that are independent of of the build tool used
```
Notice:
- Setting `docker` or `kaniko` will define the build tool for this image.
- You **cannot** use `docker` and `kaniko` in combination. 
- If neither `docker` nor `kaniko` is specified, `docker` will be used by default.

### images[*].build.docker
```yaml
docker:                             # struct   | Options for building images with Docker
  preferMinikube: true              # bool     | If available, use minikube's in-built docker daemon instaed of local docker daemon (default: true)
```

### images[*].build.kaniko
```yaml
kaniko:                             # struct   | Options for building images with kaniko
  cache: true                       # bool     | Use caching for kaniko build process
  namespace: ""                     # string   | Kubernetes namespace to run kaniko build pod in (Default: "" = deployment namespace)
  pullSecret: ""                    # string   | Mount this Kubernetes secret instead of creating one to authenticate to the registry (default: "")
```
> It is recommended to use Docker for building images when using DevSpace.cloud.

### images[*].build.options
```yaml
build:                              # struct   | Options for building images
  target: ""                        # string   | Target used for multi-stage builds
  network: ""                       # string   | Network mode used for building the image
  buildArgs: {}                     # map[string]string | Key-value map specifying build arguments that will be passed to the build tool (e.g. docker)
```


---
## deployments
```yaml
deployments:                        # struct[] | Array of deployments
- name: my-deployment               # string   | Name of the deployment
  namespace: ""                     # string   | Namespace to deploy to (Default: "" = namespace of the active Space)
  helm: ...                         # struct   | Use Helm as deployment tool and set options for Helm
  kubectl: ...                      # struct   | Use "kubectl apply" as deployment tool and set options for kubectl
```
Notice:
- Setting `helm` or `kubectl` will define the deployment tool to be used.
- You **cannot** use `helm` and `kubectl` in combination. 
- If neigther `helm` nor `kubectl` is specified, `helm` will be used by default.

### deployments[*].helm
```yaml
helm:                               # struct   | Options for deploying with Helm
  chartPath: ./chart                # string   | Relative path 
  wait: true                        # bool     | Wait for pods to start after deployment (Default: true)
  tillerNamespace: ""               # string   | Kubernetes namespace to run Tiller in (Default: "" = same a deployment namespace)
  overrideValues: {}                # struct   | Any object with Helm values to override values.yaml during deployment
  overrides: ...
```
[Learn more about configuring deployments with Helm.](../deployment/charts)

### deployments[*].kubectl
```yaml
kubectl:                            # struct   | Options for deploying with "kubectl apply"
  cmdPath: ""                       # string   | Path to the kubectl binary (Default: "" = detect automatically)
  manifests: []                     # string[] | Array containing glob patterns for the Kubernetes manifests to deploy using "kubectl apply" (e.g. kube/* or manifests/service.yaml)
```
> **It is recommended to use Helm for deployment.** To add existing manifests, you can 
[use the DevSpace helm chart](../charts/devspace-chart) and then
[add custom Kubernetes manifests](../charts/custom-manifests).


---
## dev
```yaml
dev:                                # struct   | Options for "devspace dev"
  autoReload: ...                   # struct   | Options for auto-reloading (i.e. re-deploying deployments and re-building images)
  overrideImages: []                # struct[] | Array of override settings for image building
  selectors: []                     # struct[] | Array of selectors used to select Kubernetes pods (used within terminal, ports and sync)
  terminal: ...                     # struct   | Options for the terminal proxy
  ports: []                         # struct[] | Array of port-forwarding settings for selected pods
  sync: []                          # struct[] | Array of file sync settings for selected pods
```
[Learn more about development with DevSpace.](../development/workflow)

### dev.autoReload
```yaml
autoReload:                         # struct   | Options for auto-reloading (i.e. re-deploying deployments and re-building images)
  paths: []                         # string[] | Array containing glob patterns of files that are watched for auto-reloading (i.e. reload when a file matching any of the patterns changes)
  deployments: []                   # string[] | Array containing names of deployments to watch for auto-reloading (i.e. reload when kubectl manifests or files within the Helm chart change)
  images: []                        # string[] | Array containing names of images to watch for auto-reloading (i.e. reload when the Dockerfile changes)
```

### dev.overrideImages
```yaml
overrideImages:                     # struct[] | Array of override settings for image building
- name: default                     # string   | Name of the image to apply this override rule to
  entrypoint: []                    # string[] | Array defining with the entrypoint that should be used instead of the entrypoint defined in the Dockerfile
```
[Learn more about entrypoint overriding.](../development/entrypoint-overrides)

### dev.selectors
```yaml
selectors:                          # struct[] | Array of selectors used to select Kubernetes pods (used within terminal, ports and sync)
- name: default                     # string   | Name of this pod selector (used to reference this selector within terminal, ports and sync)
  namespace: ""                     # string   | Namespace to select pods in (Default: "" = namespace of the active Space)
  ContainerName: ""                 # string   | Name of the container within the selected pod (Default: "" = first container in the pod)
  labelSelector: {}                 # map[string]string | Key-value map of Kubernetes labels used to select pods
```

### dev.terminal
```yaml
terminal:                           # struct   | Options for the terminal proxy
  selector:                         # TODO
  disabled: false                   # bool     | Disable terminal proxy / only start port-forwarding and code sync if defined (Default: false)
  command: []                       # string[] | Array defining the shell command to start the terminal with (Default: ["sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"])
```
[Learn more about configuring the terminal proxy.](../development/terminal)

### dev.ports
```yaml
ports:                              # struct[] | Array of port forwarding settings for selected pods
- selector:                         # TODO
  portMappings:                     # struct[] | Array of port mappings
  - localPort: 8080                 # int      | Forward this port on your local computer
    remotePort: 3000                # int      | Forward traffic to this port exposed by the pod selected by "selector" (TODO)
    bindAddress: ""                 # string   | Address used for binding / use 0.0.0.0 to bind on all interfaces (Default: "localhost" = 127.0.0.1)
```
[Learn more about port forwarding.](../development/port-forwarding)

### dev.sync
```yaml
sync:                               # struct[] | Array of file sync settings for selected pods
- selector:                         # TODO
  localSubPath: ./                  # string   | Relative path to a local folder that should be synchronized (Default: "./" = entire project)
  containerPath: /app               # string   | Absolute path in the container that should be synchronized with localSubPath
  excludePaths: []                  # string[] | Paths to exclude files/folders from sync in .gitignore syntax
  downloadExcludePaths: []          # string[] | Paths to exclude files/folders from download in .gitignore syntax
  uploadExcludePaths: []            # string[] | Paths to exclude files/folders from upload in .gitignore syntax
  bandwidthLimits:                  # struct   | Bandwidth limits for the synchronization algorithm
    download: 0                     # int64    | Max file download speed in kilobytes / second (e.g. 100 means 100 KB/s)
    upload: 0                       # int64    | Max file upload speed in kilobytes / second (e.g. 100 means 100 KB/s)
```
[Learn more about confguring the code synchronization.](../development/synchronization)


---
## cluster
> **Warning:** Change the cluster configuration only if you *really* know what you are doing. Editing this configuration can lead to issues with when running DevSpace.cli commands.

### using DevSpace.cloud (Enterprise)
```yaml
cluster:                            # struct   | Cluster configuration
  cloudProvider: app.devspace.cloud # string   | URL of the DevSpace.cloud instance your DevSpace.cli client is connecting to
```

### without DevSpace.cloud
> If you want to work with self-managed Kubernetes clusters, it is highly recommended to [connect an external cluster to DevSpace.cloud or run your own instance of DevSpace.cloud](../advanced/external-clusters) instead of using the following configuration options.

```yaml
cluster:                            # struct   | Cluster configuration
  kubeContext: ""                   # string   | Name of the Kubernetes context to use (Default: "" = current Kubernetes context used by kubectl)
  namespace: ""                     # string   | Namespace for deploying applications
  apiServer: ""                     # string   | URL of your Kubernetes API server (master)
  caCert: ""                        # string   | CA Certificate of your Kubernetes API server
  user:                             # struct   | Options for user authentication
    clientCert: ""                  # string   | Use certificate-based authentication using this client certificate
    clientKey: ""                   # string   | Use certificate-based authentication using this client key
    token: ""                       # string   | Use token-based authentication using this token
```
Notice:
- You **cannot** use any of these configuration options in combination with `cloudProvider`.
- You **cannot** use `clientCert` and `clientKey` in combination with `token`.
