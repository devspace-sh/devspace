---
title: Image specification
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
[Learn more about building images with DevSpace.](/docs/image-building/overview)

## images[*].build
```yaml
build:                              # struct   | Build configuration for an image
  disabled: false                   # bool     | Disable image building (Default: false)
  dockerfile: ./Dockerfile          # string   | Relative path to the Dockerfile used for building (Default: ./Dockerfile)
  context: ./                       # string   | Relative path to the context used for building (Default: ./)
  kaniko: ...                       # struct   | Build image with kaniko and set options for kaniko
  docker: ...                       # struct   | Build image with docker and set options for docker
  options: ...                      # struct   | Set build options that are independent of of the build tool used
```
Notice:
- Setting `docker` or `kaniko` will define the build tool for this image.
- You **cannot** use `docker` and `kaniko` in combination. 
- If neither `docker` nor `kaniko` is specified, `docker` will be used by default.

## images[*].build.docker
```yaml
docker:                             # struct   | Options for building images with Docker
  preferMinikube: true              # bool     | If available, use minikube's in-built docker daemon instaed of local docker daemon (default: true)
```

## images[*].build.kaniko
```yaml
kaniko:                             # struct   | Options for building images with kaniko
  cache: true                       # bool     | Use caching for kaniko build process
  snapshotMode: "time"              # string   | Type of snapshotMode for kaniko build process (compresses layers)
  namespace: ""                     # string   | Kubernetes namespace to run kaniko build pod in (Default: "" = deployment namespace)
  pullSecret: ""                    # string   | Mount this Kubernetes secret instead of creating one to authenticate to the registry (default: "")
```
> It is recommended to use Docker for building images when using DevSpace Cloud.

## images[*].build.options
```yaml
build:                              # struct   | Options for building images
  target: ""                        # string   | Target used for multi-stage builds
  network: ""                       # string   | Network mode used for building the image
  buildArgs: {}                     # map[string]string | Key-value map specifying build arguments that will be passed to the build tool (e.g. docker)
```
