---
title: /.devspace/private.yaml
---

This is an example of the `.devspace/private.yaml`

```yaml
version: v1
release:
  name: devspace-docs
  namespace: devspace-docs
  latestBuild: "2018-08-27T17:32:42.5568625+02:00"
  latestImage: 10.99.117.105:5000/devspace-docs:vkY31ug8IQ
registry:
  release:
    name: devspace-registry
    namespace: devspace-docs
  user:
    username: user-xNAsk
    password: tY2XpapTMUjt
cluster:
  tillerNamespace: devspace-docs
  useKubeConfig: true
```

The `.devspace/private.yaml` is defined for every developer that wants to use a DevSpace for working on the respective project. This file should **never** be checked into a version control system. Therefore, the DevSpace CLI will automatically create a `.gitignore` file within `.devspace/` that tells git not to version this file.

**Note: You can easily re-configure your DevSpace by running `devspace init -r`. Therefore, changing this file manually is highly discouraged.**

## Release Config
The `release` field specifies:
- `name` of the DevSpace
- `namespace` to start the DevSpace in
- `latestBuild` (time of the latest image build process, i.e. docker build)
- `latestImage` (name and tag of the latest image built by the DevSpace CLI)

## Registry Config
The `registry` field specifies:
- `release` details (`name`, `namespaces`) for deploying the internal image registry
- `user` credentials (`username`, `password`) for pushing to / pulling from the registry

## Cluster / Kubernetes Config
The `cluster` field specifies:
- `tillerNamespace` to run the Tiller server in
- `useKubeConfig` (yes to use the credentials defined in $HOME/.kube/config)

If `useKubeConfig: false` is used, the following fields needs to be specified:
- `apiServer` (Kubernetes API-Server URL)
- `caCert` (CaCert for the Kubernetes API-Server in PEM format)
- `user` specifying the following: 
  - `username`
  - `clientCert` (PEM format)
  - `clientKey` (PEM format)
