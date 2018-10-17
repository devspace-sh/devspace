---
title: /.devspace/config.yaml
---

This is a basic example of a .devspace/config.yaml. For a complete annotated version see below.
```yaml
version: v1alpha1
cluster:
  # Use the free devspace-cloud as a deploy target
  cloudProvider: devspace-cloud
devSpace:
  deployments:
  - name: devspace-default
    # For this deployment we use helm as deployment method (kubectl would be also an option)
    helm:
      chartPath: ./chart
  sync:
  - containerPath: /app
    labelSelector:
      release: devspace-default
    localSubPath: ./
    uploadExcludePaths:
    - .devspace/
images:
  default:
    name: mydockername/devspace
```

# Config reference

A [.devspace/config.yaml](#) contains any public/shared configuration for running a DevSpace for the respective project. It is highly recommended to put this file under version control (e.g. git add).

**Note: You can easily re-configure your DevSpace by running `devspace init -r`.**

## devspace
Defines the DevSpace including everything related to terminal, portForwarding, sync, and deployments.

### devspace.deployments[]
In this section, so called deployments are defined, which will be deployed to the target cluster on `devspace up`.
- `name` *string* the name of the deployment (if using helm as deployment method, also the release name)
- `namespace` *string* the namespace to deploy to
- `helm` *HelmConfig* if set, helm will be used as deployment method
- `kubectl` *KubectlConfig* if set, kubectl apply will be used as deployment method

### devspace.deployments[].helm
When specifying helm as deployment method, `devspace up` will deploy the specified chart in the target cluster. If no tiller server is found, it will also attempt to deploy a tiller server. 
- `chartPath` *string* the path where the helm chart is laying

### devspace.deployments[].kubectl
When using kubectl as deployment method, `devspace up` will use kubectl apply on the specified manifests to deploy them to the target cluster. [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) is needed in order for this option to work.  
- `cmdPath` *string* Optional: the path to the kubectl executable
- `manifests` *string array* glob patterns where the kubernetes yaml files lie (e.g. kube/* or kube/pod.yaml)

### devspace.terminal
In this section options are defined, what should happen when devspace up or devspace enter try to open a terminal. By default, devspace will select pods with the labels `release=devspace-default` and try to start a bash or sh terminal in the container.
- `namespace` *string* the namespace where to select pods from
- `labelSelector` *map[string]string* a key value map with the labels to select the correct pod (default: release: devspace-default)
- `containerName` *string* the name of the container to connect to within the selected pod (default is the first defined container)  
- `command` *string array* the default command that is executed when entering a pod with devspace up or devspace enter (default is: ["sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"])  

### devspace.ports
To access applications running inside a DevSpace, the DevSpace CLI allows to configure port forwardings. A port forwarding consists of the following:
- `namespace` *string* the namespace where to select the pods from
- `labelSelector` *map[string]string* a key value map with the labels to select from (default: release: devspace-default)
- `portMappings` *PortMapping array* 

### devspace.ports[].portMappings[]
PortMapping:
- `localPort` *string* the local port on the machine 
- `remotePort` *string* the remote pod port

In the example above, you could open `localhost:8080` inside your browser to see the output of the application listening on port 80 within your DevSpace.

### devspace.sync[]
To comfortably sync code to a DevSpace, the DevSpace CLI allows to configure real-time code synchronizations. A sync config consists of the following:
- `labelSelector` *map[string]string* a key value map with the labels to select the correct pod (default: release: devspace-default)
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
- `name` *string* name of the image with registry url prefixed (e.g. dockerhubname/image, gcr.io/googleprojectname/image etc.)
- `registry` *string* Optional: registry references one of the keys defined in the `registries` map. If defined do not prefix the image name with the registry url
- `build` *BuildConfig* defines the build procedure for this image  

### images[].build
BuildConfig:
- `dockerfilePath` *string* specifies the path where the dockerfile lies (default: ./Dockerfile)
- `contextPath` *string* specifies the context path for docker (default: ./)
- `docker` *DockerConfig* use the local Docker daemon or a Docker daemon running inside a Minikube cluster (if `preferMinikube` == true)
- `kaniko` *KanikoConfig* build images in userspace within a build pod running inside the Kubernetes cluster 
- `options` *BuildOptions* additional options used for building the image

### images[].build.docker
DockerConfig:
- `preferMinikube` *bool* if true and the current kubectl context is minikube, the minikube docker daemon is used for image building  

### images[].build.kaniko
KanikoConfig:
- `cache` *bool* if true the last image build is used as cache repository
- `namespace` *string* specifies the namespace where the build pod should be started

### images[].build.options
BuildOptions:
- `buildArgs` *map[string]string* key-value map used for specifying build arguments passed to docker
- `target` *string* the target used for multi-stage builds (see [multi-stage-build](https://docs.docker.com/develop/develop-images/multistage-build/))
- `network` *string* the network mode used for building the image (see [network](https://docs.docker.com/network/)
 
## registries
This section of the config defines a map of image registries. Use this only if you want to add authentification options to the config, otherwise just prefix the image name with the registry url. You can define in this section any external registry or link to the internalRegistry.

### registries[]
ImageRegistry:
- `url` *string* the url of the registry (format: myregistry.com:port)
- `insecure` *bool* flag to allow pushing to registries without HTTPS
- `user` *RegistryUser* credentials for pushing to / pulling from the registry

### registries[].user
RegistryUser:
- `username` *string* the user that should be used for pushing and pulling from the registry
- `password` *string* the password should be used for pushing and pulling from the registry

### internalRegistry
If devspace should deploy an internal registry for you, you can define it in this section. This is only tested with minikube and enables full offline development:
- `deploy` *bool* if the internal registry should be automatically deployed
- `namespace` *string* the namespace where to deploy the internal registry

### tiller
In this section you can define additional settings for connecting to the tiller server (if helm should be used for deployment)
- `namespace` *string* the namespace where the tiller is running (if tiller is not found, it will be deployed automatically)

## cluster
The `cluster` field specifies:
- `kubeContext` *string* the kubernetes context to use from $HOME/.kube/config
- `cloudProvider` *string* the cloud provider to use to automatically create a devspace namespace (currently only 'devspace-cloud' is supported)
- `namespace` *string* the default namespace that should be used (will override the namespace in the kubernetes context)
- `apiServer` *string* Kubernetes API-Server URL
- `caCert` *string* CaCert for the Kubernetes API-Server in PEM format
- `user`  *ClusterUser*  

### cluster.user
ClusterUser:
- `clientCert` *string* (PEM format)
- `clientKey` *string* (PEM format)  
- `token` *string* Token string for service accounts

# Full annotated config.yaml

This is a complete annotated example of a [.devspace/config.yaml](#)
```yaml
# Devspace version, currently is always v1
version: v1alpha1
# Target cluster configuration
cluster:
  # Use the local kubectl client config with context minikube
  kubeContext: minikube
  # Use this namespace as default namespace
  namespace: devspace
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
    # Label selector to select the correct pods
    labelSelector:
      release: devspace-default
  deployments:
  - name: devspace-default # this is also the release name, when using helm as deployment method
    helm:
      # Use helm to deploy this chart
      chartPath: chart/
  - name: devspace-kubectl
    kubectl: 
      manifests:
      # Use kubectl apply to deploy these manifests during `devspace up`. Devspace will also automatically append  
      # the image tag on images specified under the images key
      - kube/pod.yaml
      - kube/additional/*
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
      release: devspace-default
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
    # Image name with prefixed docker image registry
    name: grc.io/devspace-user/devspace
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
      docker:
        # Use the minikube docker daemon, if the current kubectl context is minikube
        preferMinikube: true
  database:
    name: devspace-user/devspace
    tag: 62i5e2p
    registry: internal
    build:
      kaniko:
        # Use kaniko within the target cluster to build the image
        # instead of local or minikube docker
        cache: true
# The registries the images should be pushed to
registries:
  # Internal registry that will be automatically deployed to the target
  # cluster if desired
  internal:
    # Auto-generated user and password
    user:
      username: user-XXXXX
      password: XXXXXXXXXX
# Optional: The deployed internal registry within the cluster
internalRegistry:
  deploy: true
# Optional: Tiller server that should be used within the cluster (only necessary if you want to use helm as deployment)
tiller:
  # if no tiller server is found in this namespace a tiller server will be automatically deployed
  namespace: tiller-server
```
