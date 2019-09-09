---
title: Configure Image Building
sidebar_label: Configuration
---

## Useful Commands

### `devspace add image`
To tell DevSpace to build an additional image, simply use the `devspace add image` command.
```bash
devspace add image database --dockerfile=./db/Dockerfile --context=./db --image=dscr.io/username/mysql
```

The command shown above would add a new image to your DevSpace configuration. The resulting configuration would look similar to this one:

```yaml
images:
  database:                         # from --name
    image: dscr.io/username/image   # from args[0]
    dockerfile: ./db/Dockerfile     # from --dockerfile
    context: ./db                   # from --context
```

### `devspace remove image`
Instead of manually removing an image from your configuration file, you can simply run:
```bash
devspace remove image database
```
This command would remove the image with name `database` from your `devspace.yaml`.


## Tagging
If you have any image defined in your `devspace.yaml`, DevSpace will tag this image after building with a random string and push it to the defined registry. DevSpace will then replace the image name with the just build tag in memory in the resources that should be deployed (kubernetes manifests, helm chart values or component values).  

There are cases where you do not want DevSpace to tag your images with a random tag and rather want more control over the tagging process. This can be accomplished with the help of [predefined configuration variables](/docs/configuration/variables#predefined-variables).  

For example you want to tag an image with the current git commit hash, your `devspace.yaml` would look like this:
```yaml
images:
  default:
    image: myrepo/devspace
    # This tag value is used for tagging the image 
    tag: ${DEVSPACE_GIT_COMMIT}
```

You can also combine several variables together:

```yaml
images:
  default:
    image: myrepo/devspace
    # This tag value is used for tagging the image 
    tag: ${DEVSPACE_USERNAME}-devspace-${DEVSPACE_GIT_COMMIT}-${DEVSPACE_RANDOM}
```

which would result in a more complex tag. For a complete overview which variables are available take a look at [predefined configuration variables](/docs/configuration/variables#predefined-variables), of course you can also mix predefined variables with environment or user defined variables to allow for more complex use cases.  



## Pull Secrets

When you push images to a private registry, you need to login to this registry beforehand (e.g. using `docker login`). When Kubernetes tries to pull images from a private registry, it also has to provide credentials to be authorized to pull images from this registry. The way to tell Kubernetes these credentials is to create a Kubernetes secret with these credentials. Such a secret is called image pull secret.

> DevSpace CLI can automatically create image pull secrets and add them to the `default` service account for images within your DevSpace configuration. You can enable this via the `createPullSecret` option in an image configuration.

Example:
```yaml
images:
  default:
    image: dscr.io/myusername/devspace
    # This tells DevSpace to create an image pull secret and add it to the default service account during devspace deploy & devspace dev
    createPullSecret: true
```

## Creating pull secrets manually
If you want to create your pull secret manually you can do this via the following command:

```bash
kubectl create secret docker-registry my-pull-secret --docker-server=[REGISTRY_URL] --docker-username=[REGISTRY_USERNAME] --docker-password=[REGISTRY_PASSWORD] --docker-email=[YOUR_EMAIL]
```

This `kubectl` command would create an image pull secret called `my-pull-secret`. 

[Learn more about image pull secrets in Kubernetes.](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)

However you also have to add it to your service account by running this command:
```bash
kubectl edit serviceaccount default
```

Then add the just created pull secret:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
  namespace: default
secrets:
- name: default-token-6k6fc
imagePullSecrets:
- name: my-pull-secret
```

Save and now you should be able to pull images from that registry.




## images
```yaml
images:                             # map[string]struct | Images to be built and pushed
  image1:                           # string   | Name of the image
    image: dscr.io/username/image   # string   | Image repository and name 
    tag: v0.0.1                     # string   | The Image tag to use for this image. See [tagging](/docs/image-building/tagging) for more information about dynamic image tags
    dockerfile: ./Dockerfile        # string   | Relative path to the Dockerfile used for building (Default: ./Dockerfile)
    context: ./                     # string   | Relative path to the context used for building (Default: ./)
    createPullSecret: true          # bool     | Create a pull secret containing your Docker credentials (Default: false)
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



TODO: ENV VARS
