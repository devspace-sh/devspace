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
      devOverwrite: ./chart/dev-overwrite.yaml
  sync:
  - containerPath: /app
    labelSelector:
      release: devspace-default
    localSubPath: ./
    uploadExcludePaths:
    - .devspace/
  portForwarding:
  - labelSelector:
      release: devspace-default
    portMappings:
    - localPort: 3000
      remotePort: 3000
images:
  default:
    name: mydockername/devspace
```

# Config reference

A [.devspace/config.yaml](#) contains any public/shared configuration for running a DevSpace for the respective project. It is highly recommended to put this file under version control (e.g. git add).

**Note: You can easily re-configure your DevSpace by running `devspace init -r`.**

## cluster
The `cluster` field specifies:
- `kubeContext` *string* the kubernetes context to use (if omitted and apiServer is not defined the current kubectl context is used)
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

## devspace
Defines the DevSpace including everything related to terminal, portForwarding, sync, and deployments:
- `deployments` *DeploymentConfig array* the deployments to deploy to the target cluster
- `services` *ServiceConfig array* DevSpace services that define common labelSelectors to use to select the correct pods
- `terminal` *TerminalConfig* terminal configuration to use for devspace up/devspace enter
- `autoReload` *AutoReloadConfig* additional paths to watch for changes to reload the build and deploy pipeline
- `ports` *PortConfig array* the ports that should be forwarded by devspace from the cluster to localhost
- `sync` *SyncConfig array* the paths that should be synced between your local machine and the remote containers

### devspace.deployments[]
In this section, so called deployments are defined, which will be deployed to the target cluster on `devspace up`.
- `name` *string* the name of the deployment (if using helm as deployment method, also the release name)
- `namespace` *string* the namespace to deploy to
- `autoReload` *AutoReloadConfig* auto reload configuration
- `helm` *HelmConfig* if set, helm will be used as deployment method
- `kubectl` *KubectlConfig* if set, kubectl apply will be used as deployment method

### devspace.deployments[].autoReload
By default devspace will reload the build and deploy process on certain changes (when helm is chosen on changes to the chart path, when kubectl on changes to the manifest paths), in this section this behaviour can be disabled
- `disabled` *bool* if true devspace does not reload the pipeline on kubectl manifest or helm chart changes

### devspace.deployments[].helm
When specifying helm as deployment method, `devspace up` will deploy the specified chart in the target cluster. If no tiller server is found, it will also attempt to deploy a tiller server. 
- `chartPath` *string* the path where the helm chart is laying
- `wait` *bool* wait till everything is ready after deployment (default: true)
- `overwrite` *string* the path to a file that overwrites the values.yaml 

### devspace.deployments[].kubectl
When using kubectl as deployment method, `devspace up` will use kubectl apply on the specified manifests to deploy them to the target cluster. [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl) is needed in order for this option to work.  
- `manifests` *string array* glob patterns where the kubernetes yaml files lie (e.g. kube/* or kube/pod.yaml)

### devspace.services[]
To define resource selectors as DevSpace services:
- `name` *string* the name of the DevSpace service
- `namespace` *string* the namespace where to select the pods from
- `labelSelector` *map[string]string* a key value map with the labels to select from (default: release: devspace-default)
- `containerName` *string* name of the container to select within the selected pod
- `resouceType` *string* Kubernetes resouce type to select (currently only `pod` is available)
These services can be referenced within other config options (e.g. terminal, ports and sync).

### devspace.terminal
In this section options are defined, what should happen when devspace up or devspace enter try to open a terminal. By default, devspace will select pods with the labels `release=devspace-default` and try to start a bash or sh terminal in the container.
- `disabled` *bool* if true no terminal will be opened on `devspace up` and devspace will try to attach to the pod instead (On failure sync & port forwarding continues)
- `service` *string* DevSpace service to start the terminal for (use either service OR namespace, labelSelector, containerName)
- `namespace` *string* the namespace where to select pods from
- `labelSelector` *map[string]string* a key value map with the labels to select the correct pod (default: release: devspace-default)
- `containerName` *string* the name of the container to connect to within the selected pod (default is the first defined container)  
- `command` *string array* the default command that is executed when entering a pod with devspace up or devspace enter (default is: ["sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"])  

### devspace.autoReload
In this section paths can be specified that should be watched by devspace for changes. If any change occurs the build and deploy pipeline is reexecuted
- `paths` *string array* path globs which devspace should watch for changes (e.g. `config/**`, `.env`, `node/package*` etc.)

### devspace.ports
To access applications running inside a DevSpace, the DevSpace CLI allows to configure port forwardings. A port forwarding consists of the following:
- `service` *string* DevSpace service to start port forwarding for (use either service OR namespace, labelSelector, resourceType)
- `namespace` *string* the namespace where to select the pods from
- `labelSelector` *map[string]string* a key value map with the labels to select from (default: release: devspace-default)
- `resouceType` *string* Kubernetes resouce type to select (currently only `pod` is available)
- `portMappings` *PortMapping array* 

### devspace.ports[].portMappings[]
PortMapping:
- `localPort` *string* the local port on the machine 
- `remotePort` *string* the remote pod port
- `bindAddress` *string* the address to bind to, optional - binds to localhost only if not present, use `0.0.0.0` for all interfaces

In the example above, you could open `localhost:8080` inside your browser to see the output of the application listening on port 80 within your DevSpace.

### devspace.sync[]
To comfortably sync code to a DevSpace, the DevSpace CLI allows to configure real-time code synchronizations. A sync config consists of the following:
- `service` *string* DevSpace service to start the sync for (use either service OR namespace, labelSelector, containerName)
- `namespace` *string* the namespace where to select the pods from
- `labelSelector` *map[string]string* a key value map with the labels to select the correct pod (default: release: devspace-default)
- `containerName` *string* the name of the container within the pod to sync to (default: the first specified container in the pod)
- `localSubPath` *string* relative path to the folder that should be synced (default: path to your local project root)
- `containerPath` *string* absolute path within the container
- `excludePaths` *string array* paths to exclude files/folders from sync in .gitignore syntax
- `downloadExcludePaths` *string array* paths to exclude files/folders from download in .gitignore syntax
- `uploadExcludePaths` *string array* paths to exclude files/folders from upload in .gitignore syntax
- `bandwidthLimits` *BandwidthLimits* the bandwidth limits to use for the syncpath

In the example above, the entire code within the project would be synchronized with the folder `/app` inside the DevSpace, with the exception of the `node_modules/` folder.

### devspace.sync[].bandwidthLimits
Bandwidth limits to use for syncing:
- `upload` *string* kilobytes per second as upper limit to use for uploading files (e.g. 100 means 100 KByte per seconds)
- `download` *string* kilobytes per second as upper limit to use for downloading files (e.g. 100 means 100 KByte per seconds)

## images
This section of the config defines a map of images that can be used in the helm chart that is deployed during `devspace up`. 

### images[]
An image is defined by:
- `name` *string* name of the image with registry url prefixed (e.g. dockerhubname/image, gcr.io/googleprojectname/image etc.)
- `createPullSecret` *bool* creates a pull secret in the cluster namespace if the credentials are available in the docker credentials store or specified under `registries[].auth`
- `registry` *string* Optional: registry references one of the keys defined in the `registries` map. If defined do not prefix the image name with the registry url
- `skipPush` *bool* if true the image push step is skipped for this image (useful for minikube setups see [minikube-example](https://github.com/covexo/devspace/tree/master/examples/minikube))
- `autoReload` *AutoReloadConfig* auto reload configuration
- `build` *BuildConfig* defines the build procedure for this image  

### images[].autoReload
By default devspace will reload the build and deploy process if the specified dockerfile is changed, in this section this behaviour can be disabled
- `disabled` *bool* if true devspace does not reload the pipeline on dockerfile changes

### images[].build
BuildConfig by default docker is used to build images:
- `dockerfilePath` *string* specifies the path where the dockerfile lies (default: ./Dockerfile)
- `contextPath` *string* specifies the context path for docker (default: ./)
- `docker` *DockerConfig* use the local Docker daemon or a Docker daemon running inside a Minikube cluster (if `preferMinikube` == true)
- `kaniko` *KanikoConfig* build images in userspace within a build pod running inside the Kubernetes cluster 
- `options` *BuildOptions* additional options used for building the image
- `disabled` *bool* Optional: if true building is skipped for this image (Can be useful when using in overwrite.yaml for users who don't have docker installed)

### images[].build.docker
DockerConfig:
- `preferMinikube` *bool* if true and the current kubectl context is minikube, the minikube docker daemon is used for image building  

### images[].build.kaniko
KanikoConfig:
- `cache` *bool* if true the last image build is used as cache repository
- `namespace` *string* specifies the namespace where the build pod should be started
- `pullSecret` *string* mount this pullSecret instead of creating one to authenticate to the registry (see [kaniko](https://github.com/covexo/devspace/tree/master/examples/kaniko) for an example)

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
- `insecure` *bool* optional: flag to allow pushing to registries without HTTPS
- `auth` *RegistryAuth* optional: credentials for pushing to / pulling from the registry (devspace automatically tries to find them in the docker credentials store)

### registries[].auth
RegistryAuth:
- `username` *string* the user that should be used for pushing and pulling from the registry
- `password` *string* the password should be used for pushing and pulling from the registry

## internalRegistry
If devspace should deploy an internal registry for you, you can define it in this section. This is only tested with minikube and enables full offline development:
- `deploy` *bool* if the internal registry should be automatically deployed
- `namespace` *string* optional: the namespace where to deploy the internal registry

## tiller
In this section you can define additional settings for connecting to the tiller server (if helm should be used for deployment)
- `namespace` *string* optional: the namespace where the tiller is running (if tiller is not found, it will be deployed automatically)

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
  # defines services of this DevSpace = pod selector
  services:
    # name of the service
  - name: default
    # namespace where to find the service (pod)
    # uses default deployment namespace if not specified
    namespace: my-namespace
    # map of labels for selecting the service pod
    labelSelector:
      devspace: default
    # container name for selecting one container of a pod 
    # (optional, uses first container if not specified)
    containerName: main-container
    # Kubernetes resource type (currently, only pod is supported)
    # (optional, uses pod if not specified)
    resourceType: pod
  # terminal options for devspace up and devspace enter
  terminal:
    # if you don't want devspace to automatically open a terminal for 
    # you set disabled to true
    disabled: false
    # define the service to start the terminal for
    service: default
    # Alternative to using a service is to 
    # specify namespace, labelSelector and containerName individually
    # Example:
  # namespace: my-namespace
  # labelSelector:
  #   devspace: default
  # containerName: main-container
    # the command to execute within the container when using `devspace up` or `devspace enter`
    command:
    - sh
    - -c
    - bash
  # Auto reload specifies on which paths the devspace up command should listen for changes. On change the command will rebuild and redeploy
  autoReload:
    paths:
    - package.json
    - config/**
    - php/php.ini
  deployments:
  - name: devspace-default # this is also the release name, when using helm as deployment method
    helm:
      # Use helm to deploy this chart
      chartPath: ./chart
      # Don't wait till everything is ready
      # wait: false
      # Overwrite the values.yaml with dev-values.yaml when running devspace up
      devOverwrite: ./chart/dev-overwrite.yaml
  - name: devspace-kubectl
    namespace: kubectl-deployment
    autoReload:
      # do not rebuild and redeploy on changes to these manifests
      disabled: true
    kubectl: 
      manifests:
      # Use kubectl apply to deploy these manifests during `devspace up`. Devspace will also automatically append  
      # the image tag on images specified under the images key
      - kube/pod.yaml
      - kube/additional/*
  # Automatically forwarded ports on `devspace up` (same functionality as running manually kubectl port-forward)
  portForwarding:
    # define the service to start port forwarding for
  - service: default
    # Alternative to using a service is to 
    # specify namespace, labelSelector and resourceType individually
    # Example:
  # namespace: my-namespace
  # labelSelector:
  #   devspace: default
  # resourceType: pod
    # Array of port mappings
    portMappings:
      # The local machine port
    - localPort: 3000
      # The selected pod port
      remotePort: 3000
    - localPort: 8080
      remotePort: 80
  sync:
    # define the service to start the sync for
  - service: default
    # Alternative to using a service is to 
    # specify namespace, labelSelector and containerName individually
    # Example:
  # namespace: my-namespace
  # labelSelector:
  #   devspace: default
  # containerName: main-container
    # Sync the complete local project path
    localSubPath: ./
    # Into the remote container path /app
    containerPath: /app
    # Exclude node_modules from up and download
    excludePaths:
    - node_modules/
    # Bandwidth limits for this sync path in Kbyte/s
    bandwidthLimits:
      # limit download speed to 100 Kbyte/s
      download: 100
      # limit upload speed to 1024 Kbyte/s
      upload: 1024
# A map of images that should be build during devspace up
images:
  default:
    # Image name with prefixed docker image registry
    name: grc.io/devspace-user/devspace
    # Specifies how to build the image
    build:
      # Specifies where the Dockerfile lies 
      dockerfilePath: ./Dockerfile
      # Specifies where the docker context path is
      contextPath: ./
      # uncomment to not rebuild and redeploy on changes to the dockerfile
      # autoReload:
      #  disabled: true
      # use docker as build engine
      docker:
        # Use the minikube docker daemon if the current kubectl context is minikube
        preferMinikube: true
      options:
        # Used for multi-stage builds
        target: development
        # buildArgs passed to docker during build
        buildArgs:
          myarg1: myvalue1
        # network mode (see [network](https://docs.docker.com/network/))
        network: bridge
  database:
    name: devspace-user/devspace
    registry: internal
    # Automatically create a pull secret for this image/registry
    createPullSecret: true
    build:
      kaniko:
        # Use kaniko within the target cluster to build the image
        # instead of local or minikube docker
        cache: true
  privateRegistryImage:
    name: user/test
    # Automatically create a pull secret for this image/registry
    createPullSecret: true
    registry: privateRegistry
# Optional: the registries the images should be pushed to
registries:
  # Internal registry that will be automatically deployed to the target
  # cluster if desired
  internal:
    # Auto-generated user and password
    auth:
      username: user-XXXXX
      password: XXXXXXXXXX
  # Private registry used by image privateRegistryImage
  privateRegistry:
    url: myPrivateRegistry.com:8080
    auth:
      username: user-XXXXX
      password: XXXXXXXXXX # Can also be a token
# Optional: Deploy internal registry within the cluster
internalRegistry:
  deploy: true
# Optional: Tiller server that should be used within the cluster (only necessary if you want to use helm as deployment)
tiller:
  # if no tiller server is found in this namespace a tiller server will be automatically deployed
  namespace: tiller-server
```
