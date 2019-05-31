---
title: Image specification
---

## images
```yaml
images:                             # map[string]struct | Images to be built and pushed
  image1:                           # string   | Name of the image
    image: dscr.io/username/image   # string   | Image repository and name 
    tag: v0.0.1                     # string   | Image tag
    dockerfile: ./Dockerfile        # string   | Relative path to the Dockerfile used for building (Default: ./Dockerfile)
    context: ./                     # string   | Relative path to the context used for building (Default: ./)
    createPullSecret: true          # bool     | Create a pull secret containing your Docker credentials (Default: true)
    build: ...                      # struct   | Build options for this image
  image2: ...
```
[Learn more about building images with DevSpace.](/docs/image-building/overview)

### images[\*].build
```yaml
build:                              # struct   | Build configuration for an image
  disabled: false                   # bool     | Disable image building (Default: false)
  kaniko: ...                       # struct   | Build image with kaniko and set options for kaniko
  docker: ...                       # struct   | Build image with docker and set options for docker
  custom: ...                       # struct   | Build image using a custom build script
```
Notice:
- Setting `docker`, `kaniko` or `custom` will define the build tool for this image.
- You **cannot** use `docker`, `kaniko` and `custom` in combination. 
- If neither `docker`, `kaniko` nor `custom` is specified, `docker` will be used by default.
- By default `docker` will use `kaniko` as fallback when DevSpace CLI is unable to reach the Docker host.

### images[\*].build.docker
```yaml
docker:                             # struct   | Options for building images with Docker
  preferMinikube: true              # bool     | If available, use minikube's in-built docker daemon instaed of local docker daemon (default: true)
  skipPush: false                   # bool     | Skip pushing image to registry, recommended for minikube (Default: false)
  disableFallback: false            # bool     | Disable using kaniko as fallback when Docker is not installed (Default: false)
  options: ...                      # struct   | Set build general build options
```

### images[\*].build.kaniko
```yaml
kaniko:                             # struct   | Options for building images with kaniko
  cache: true                       # bool     | Use caching for kaniko build process
  snapshotMode: "time"              # string   | Type of snapshotMode for kaniko build process (compresses layers)
  flags: []                         # string[] | Array of flags for kaniko build command
  namespace: ""                     # string   | Kubernetes namespace to run kaniko build pod in (Default: "" = deployment namespace)
  insecure: false                   # bool     | Allow working with an insecure registry by not validating the SSL certificate (Default: false)
  pullSecret: ""                    # string   | Mount this Kubernetes secret instead of creating one to authenticate to the registry (default: "")
  options: ...                      # struct   | Set build general build options
```

### images[\*].build.custom
```yaml
custom:                             # struct   | Options for building images with a custom build script
  command: "./scripts/builder"      # string   | Path to the build script
  flags: []                         # string[] | Array of flags for the build script
  imageFlag: string                 # string   | Name of the flag that DevSpace CLI uses to pass the image name + tag to the build script
  onChange: []                      # string[] | Array of paths (glob format) to check for file changes to see if image needs to be rebuild
```

### images[\*].build.\*.options
```yaml
options:                            # struct   | Options for building images
  target: ""                        # string   | Target used for multi-stage builds
  network: ""                       # string   | Network mode used for building the image
  buildArgs: {}                     # map[string]string | Key-value map specifying build arguments that will be passed to the build tool (e.g. docker)
```
