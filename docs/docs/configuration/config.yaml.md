---
title: /.devspace/config.yaml
---

This is an example of a [.devspace/config.yaml](#)
```yaml
version: v1
devSpace:
  release:
    name: my-project
    namespace: my-namespace
  portForwarding:
  - resourceType: pod
    labelSelector:
      release: my-app
    portMappings:
    - localPort: 3000
      remotePort: 3000
    - localPort: 8080
      remotePort: 80
  sync:
  - resourceType: pod
    labelSelector:
      release: my-app
    localSubPath: ./
    containerPath: /app
images:
  default:
    name: devspace-user/devspace
    tag: 9u5ye0G
    registry: default
    build:
      engine:
        docker:
          enabled: true
          preferMinikube: true
  database:
    name: devspace-user/devspace
    tag: 62i5e2p
    registry: internal
    build:
      engine:
        kaniko:
          enabled: true
registries:
  default:
    url: hub.docker.com
  internal:
    user:
      username: user-XXXXX
      password: XXXXXXXXXX
services:
  internalRegistry:
    release:
      name: devspace-registry
      namespace: my-namespace
  tiller:
    appNamespaces:
    - my-namespace
    release:
      namespace: my-namespace
cluster:
  useKubeConfig: true
```
A [.devspace/config.yaml](#) contains any public/shared configuration for running a DevSpace for the respective project. It is highly recommended to put this file under version control (e.g. git add).

**Note: You can easily re-configure your DevSpace by running `devspace init -r`.**

## devspace
Defines your DevSpace including everything related to portForwarding, sync, and the release config.

### devspace.release
Defines how the DevSpace is deployed to your cluster. See [Type: Release](#type-release) for details.

### devspace.portForwarding
To access applications running inside a DevSpace, the DevSpace CLI allows to configure port forwardings. A port forwarding consists of the following:
- `resourceType` (currently only `pod` is supported)
- `labelSelector` (usually the release/app name)
- a list of `portMappings` (each specifying a `localPort` on localhost and a `remotePort` within the DevSpace)

In the example above, you could open `localhost:8080` inside your browser to see the output of the application listening on port 80 within your DevSpace.

### devspace.sync
To comfortably sync code to a DevSpace, the DevSpace CLI allows to configure real-time code synchronizations. A sync config consists of the following:
- `resourceType` (currently only `pod` is supported)
- `labelSelector` (usually the release/app name)
- `localSubPath` (relative to your local project root)
- `containerPath` (absolute path within your DevSpace)
- `excludePaths` (for excluding files/folders from sync in .gitignore syntax)
- `DownloadExcludePaths` (for excluding files/folders from download in .gitignore syntax)
- `UploadExcludePaths` (for excluding files/folders from upload in .gitignore syntax)

In the example above, the entire code within the project would be synchronized with the folder `/app` inside the DevSpace.

## images
This section of the config defines a map of images that can be used in the helm chart that is deployed during `devspace up`. An image is defined by:
- `name` of the image that is being pushed to the registry
- `tag` stating the latest tag pushed to the registry
- `registry` referencing one of the keys defined in the `registries` map
- `build` defining the build procedure for this image

## images[*].build
An image build is mainly defined by the build engine. There are 2 build engines currently supported:
- `docker` uses the local Docker daemon or a Docker daemon running inside a Minikube cluster (if `preferMinikube` == true)
- `kaniko` builds images in userspace within a build pod running inside the Kubernetes cluster

## registries
This section of the config defines a map of image registries. You can use any external registry or link to the [services.internalRegistry](#services-internal-registry)
- `url` of the registry (format: myregistry.com:port)
- `user` credentials (`username`, `password`) for pushing to / pulling from the registry
- `insecure` flag to allow pushing to registries without HTTPS

## services
Defines additional services for your DevSpace.

### services.internalRegistry
The `internalRegistry` is used to tell the DevSpace CLI to deploy a private registry inside the Kubernetes cluster:
- `release` for deploying the registry (see [Type: Release](#type-release))

### services.tiller
The `tiller` service is defined by:
- `release` definition for tiller (see [Type: Release](#type-release))
- `appNamespaces` defining a list of namespace that tiller may deploy applications to

## cluster
The `cluster` field specifies:
- `useKubeConfig` (yes to use the credentials defined in $HOME/.kube/config)

If `useKubeConfig: false` is used, the following fields need to be specified:
- `apiServer` (Kubernetes API-Server URL)
- `caCert` (CaCert for the Kubernetes API-Server in PEM format)
- `user` specifying the following: 
  - `username`
  - `clientCert` (PEM format)
  - `clientKey` (PEM format)

## Type: Release
A `release` is specified through:
- `name` of the release
- `namespace` to deploy the release to
- `values` that are set during the deployment (contents of the values.yaml in helm)
