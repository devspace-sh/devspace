---
title: /.devspace/config.yaml
---

This is an example of a [.devspace/config.yaml](#)
```yaml
# Devspace version, currently is always v1
version: v1
devSpace:
  # terminal options for devspace up and devspace enter
  terminal:
    # the container name within the selected release pod to open a terminal connection to (is also a flag in `devspace up -c CONTAINER`)
    containerName: default
    # the command to execute within the container when using `devspace up` or `devspace enter`
    command:
    - sh
    - -c
    - bash
  release:
    # Name of helm release that is used for deploying
    # the devspace chart (contents of /chart)
    name: my-project
    # Release namespace
    namespace: my-namespace
  # Automatically forwarded ports on `devspace up` (same functionality as running manually kubectl port-forward)
  portForwarding:
    # Currently only pod is supported
  - resourceType: pod
    # Map of key value matchLabel selectors
    labelSelector:
      release: my-app
    # namespace where to select the pods from
    namespace: my-namespace
    # Array of port mappings
    portMappings:
      # The local machine port
    - localPort: 3000
      # The selected pod port
      remotePort: 3000
    - localPort: 8080
      remotePort: 80
  sync:
    # Currently only resource type pod is supported
  - resourceType: pod
    labelSelector:
      release: my-app
    # The container within the pod to sync to
    containerName: default
    # Sync the complete local project path
    localSubPath: ./
    # Into the remote container path /app
    containerPath: /app
    # Exclude node_modules from up and download
    excludePaths:
    - node_modules/
# A list of images that should be build during devspace up
images:
  default:
    # Image name
    name: devspace-user/devspace
    # Image tag (auto-generated)
    tag: 9u5ye0G
    # Registry where the build image will be pushed to
    registry: default
    # Specifies where the docker context path is
    contextPath: ./
    # Specifies where the Dockerfile lies 
    dockerfilePath: ./Dockerfile
    # Specifies how to build the image
    build:
      options:
        # Used for multi-stage builds
        target: development
        # buildArgs passed to docker during build
        buildArgs:
          myarg1: myvalue1
        # network mode (see [network](https://docs.docker.com/network/))
        network: bridge
      engine:
        docker:
          # Use docker for image building
          enabled: true
          # Use the minikube docker daemon, if the current kubectl context is minikube
          preferMinikube: true
  database:
    name: devspace-user/devspace
    tag: 62i5e2p
    registry: internal
    build:
      engine:
        kaniko:
          # Use kaniko within the target cluster to build the image
          # instead of local or minikube docker
          enabled: true
# The registries the images should be pushed to
registries:
  default:
    url: hub.docker.com
  # Internal registry that will be automatically deployed to the target
  # cluster if desired
  internal:
    # Auto-generated user and password
    user:
      username: user-XXXXX
      password: XXXXXXXXXX
# Services that are used within the target cluster
services:
  # The deployed internal registry within the cluster
  internalRegistry:
    release:
      # The helm release name of the internal registry
      name: devspace-registry
      namespace: my-namespace
  # Tiller server that should be used within the cluster
  tiller:
    # A list of namespaces where tiller is allowed to deploy namespaces to
    appNamespaces:
    - my-namespace
    release:
      # Namespace where the tiller server is located
      namespace: my-namespace
# Target cluster configuration
cluster:
  # Use the local kubectl client config
  useKubeConfig: true
```
A [.devspace/config.yaml](#) contains any public/shared configuration for running a DevSpace for the respective project. It is highly recommended to put this file under version control (e.g. git add).

**Note: You can easily re-configure your DevSpace by running `devspace init -r`.**

## devspace
Defines the DevSpace including everything related to terminal, portForwarding, sync, and the helm release config.

### devspace.terminal
In this section options are defined, what should happen when devspace up or devspace enter connect to the release pod.
- `containerName` *string* the name of the container to connect to within the selected pod (default is the first defined container)
- `command` *string array* the default command that is executed when entering a pod with devspace up or devspace enter (default is: ["sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"])

### devspace.release
Defines how the DevSpace is deployed to your cluster. See [Type: Release](#type-release) for details.

### devspace.portForwarding
To access applications running inside a DevSpace, the DevSpace CLI allows to configure port forwardings. A port forwarding consists of the following:
- `resourceType` *string* kubernetes resource type that is selected (currently only `pod` is supported)
- `namespace` *string* the namespace where to select the pods from
- `labelSelector` *map[string]string* usually the release/app name
- `portMappings` *PortMapping array* 

### devspace.portForwarding[].portMappings[]
PortMapping:
- `localPort` *string* the local port on the machine 
- `remotePort` *string* the remote pod port

In the example above, you could open `localhost:8080` inside your browser to see the output of the application listening on port 80 within your DevSpace.

### devspace.sync
To comfortably sync code to a DevSpace, the DevSpace CLI allows to configure real-time code synchronizations. A sync config consists of the following:
- `resourceType` *string* kubernetes resource type that is selected (currently only `pod` is supported)
- `labelSelector` *map[string]string* label selector to select the correct pod (usually the release/app name)
- `namespace` *string* the namespace where to select the pods from
- `containerName` *string* the name of the container within the pod to sync to (default: the first specified container in the pod)
- `localSubPath` *string* relative path to the folder that should be synced (default: path to your local project root)
- `containerPath` *string* absolute path within the container
- `excludePaths` *string array* paths to exclude files/folders from sync in .gitignore syntax
- `downloadExcludePaths` *string array* paths to exclude files/folders from download in .gitignore syntax
- `uploadExcludePaths` *string array* paths to exclude files/folders from upload in .gitignore syntax

In the example above, the entire code within the project would be synchronized with the folder `/app` inside the DevSpace, with the exception of the `node_modules/` folder.

## images
This section of the config defines a map of images that can be used in the helm chart that is deployed during `devspace up`. 

### images[]
An image is defined by:
- `name` *string* name of the image that is being pushed to the registry
- `tag` *string* tag indicates the latest tag pushed to the registry (auto-generated)
- `registry` *string* registry references one of the keys defined in the `registries` map
- `build` *BuildConfig* defines the build procedure for this image  

### images[].build
BuildConfig:
- `dockerfilePath` *string* specifies the path where the dockerfile lies (default: ./Dockerfile)
- `contextPath` *string* specifies the context path for docker (default: ./)
- `engine` *Engine* the engine that should be used for building the image  
- `options` *BuildOptions* additional options used for building the image

### images[].build.options
BuildOptions:
- `buildArgs` *map[string]string* key-value map used for specifying build arguments passed to docker
- `target` *string* the target used for multi-stage builds (see [multi-stage-build](https://docs.docker.com/develop/develop-images/multistage-build/))
- `network` *string* the network mode used for building the image (see [network](https://docs.docker.com/network/))

### images[].build.engine
Engine:
An image build is mainly defined by the build engine. There are 2 build engines currently supported (choose only one):
- `docker` *DockerConfig* use the local Docker daemon or a Docker daemon running inside a Minikube cluster (if `preferMinikube` == true)
- `kaniko` *KanikoConfig* build images in userspace within a build pod running inside the Kubernetes cluster  

### images[].build.engine.docker
DockerConfig:
- `enabled` *bool* if true the local docker daemon is used for image building
- `preferMinikube` *bool* if true and the current kubectl context is minikube, the minikube docker daemon is used for image building  

### images[].build.engine.kaniko
KanikoConfig:
- `enabled` *bool* if true a kaniko build pod is used for image building
- `namespace` *string* specifies the namespace where the build pod should be started  

## registries
This section of the config defines a map of image registries. You can use any external registry or link to the [services.internalRegistry](#services-internal-registry)

### registries[]
ImageRegistry:
- `url` *string* the url of the registry (format: myregistry.com:port)
- `insecure` *bool* flag to allow pushing to registries without HTTPS
- `user` *RegistryUser* credentials for pushing to / pulling from the registry

### registries[].user
RegistryUser:
- `username` *string* the user that should be used for pushing and pulling from the registry
- `password` *string* the password should be used for pushing and pulling from the registry

## services
Defines cluster services that the DevSpace uses.

### services.internalRegistry
The `internalRegistry` is used to tell the DevSpace CLI to deploy a private registry inside the Kubernetes cluster:
- `release` *Release* release options for deploying the registry (see [Type: Release](#type-release))

### services.tiller
The `tiller` service is defined by:
- `release` *Release* release definition for tiller (see [Type: Release](#type-release))
- `appNamespaces` *string array* defines a list of namespace that tiller may deploy applications to  

## cluster
The `cluster` field specifies:
- `useKubeConfig` *bool* if true use the credentials defined in $HOME/.kube/config
- `kubeContext` *string* the context to use from $HOME/.kube/config

If `useKubeConfig` is `false`, the following fields need to be specified:
- `apiServer` *string* (Kubernetes API-Server URL)
- `caCert` *string* (CaCert for the Kubernetes API-Server in PEM format)
- `user`  *ClusterUser*  

### cluster.user
ClusterUser:
- `clientCert` *string* (PEM format)
- `clientKey` *string* (PEM format)  

## Type: Release
A `release` is specified through:
- `name` *string* name of the release
- `namespace` *string* the namespace to deploy the release to
- `values` *map[string] any* override values that are set during the deployment (contents of the values.yaml in helm)  
