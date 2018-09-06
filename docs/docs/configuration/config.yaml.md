---
title: /.devspace/config.yaml
---

This is an example of a [.devspace/config.yaml](#)
```yaml
version: v1
devSpace:
  release:
    name: devspace-cloud-com
    namespace: dev-gentele
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
image:
  name: devspace
services:
  registry:
    internal:
      release:
        name: devspace-registry
        namespace: dev-gentele
    user:
      username: user-XXXXX
      password: XXXXXXXXXX
  tiller:
    release:
      namespace: dev-gentele
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

## image
An image is defined by:
- `name` of the image that is being pushed to the registry
- `tag` stating the latest tag pushed to the registry
- `buildTime` (time of the latest image build process, i.e. docker build)

## services
Defines additional services for your DevSpace.

### services.registry
The `registry` field specifies:
- `external` tells the DevSpace CLI to push to an external registry (format: myregistry.com:port)
- `internal` defines a private cluster-internal registry by defining a `release` for it
- `user` credentials (`username`, `password`) for pushing to / pulling from the registry
- `insecure` flag to allow pushing to registries without HTTPS

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
