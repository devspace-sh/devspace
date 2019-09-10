---
title: Configuring Build Tools
sidebar_label: Build Tools
---

The `build` option of each image (under `images`) defines which build tool DevSpace uses to build the image. The following build tools are currently supported:
- [`docker`](#docker) for building images using a Docker daemon (**default build tool**, [prefers Docker daemon of local Kubernetes clusters](/docs/cli/image-building/workflow-basics#docker-daemon-of-local-kubernetes-clusters))
- [`kaniko`](#kaniko) for building images directly inside Kubernetes ([fallback for `docker`](#dockerdisablefallback-kaniko-as-fallback-for-docker))
- [`custom`](#custom) for building images with a custom build command (e.g. for using Google Cloud Build)
- [`disabled`](#disabled) for disabling image building for this image

> Different images can be built using different build tools.


## `docker` (default)
If nothing is specified, DevSpace always tries to build the image using `docker` as build tool.

### `docker.disableFallback` *Kaniko as Fallback for Docker*
When using `docker` as build tool, DevSpace checks if Docker is installed and running. If Docker is not installed or not running, DevSpace will use kaniko as fallback to build the image unless the option [`disableFallback`](#dockerdisablefallback-kaniko-as-fallback-for-docker) is set to `false`.

#### Default Value For `disableFallback`
```yaml
disableFallback: false
```

#### Example: Disabling kaniko Fallback
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    build:
      docker:
        disableFallback: true
```
**Explanation:**  
- The first image `backend` would be built using Docker and if Docker is not available, the image would be built using kaniko as a fallback.
- The second image `frontend` would be built using Docker and if Docker is not available, DevSpace would exit with a fatal error instead of using the kaniko fallback.

### `docker.preferMinikube` *Building Images in Minikube*
DevSpace preferably uses the Docker daemon running in the virtual machine that belongs to your local Kubernetes cluster instead of your regular Docker daemon. This has the advantage that images do not need to be pushed to a registry because Kubernetes can simply use the images available in the Docker daemon belonging to the kubelet of the local cluster. Using this method is only possible when your current kube-context points to a local Kubernetes cluster and is named `minikube`, `docker-desktop` or `docker-for-desktop`.

#### Default Value For `preferMinikube`
```yaml
preferMinikube: true
```

#### Example: Building Images in Minikube
```yaml
images:
  backend:
    image: john/appbackend
    build:
      docker:
        preferMinikube: true
        skipPush: true
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    build:
      docker:
        preferMinikube: false
```
**Explanation:**  
- The first image `backend` would be preferably built with Minikube's Docker daemon and the image would **not** be pushed to a registry.
- The second image `frontend` would **not** be built with the Docker daemon of Minikube and it would be pushed to a registry after building and tagging the image using Docker (or kaniko as fallback).

### `docker.skipPush`
The `skipPush` option expects a boolean value stating if pushing the image to a registry should be skipped during the build process.

If DevSpace is using a local Kubernetes cluster (e.g. minikube or Docker Kubernetes), pushing images might not be necessary because the image might already be accessible by Kubernetes via a local Docker daemon. In this case, it can make sense to speed up the process by setting `skipPush` to `true`.

#### Default Value For `skipPush`
```yaml
skipPush: false
```

#### Example
**See "[Example: Building Images in Minikube](#example-building-images-in-minikube)"**


### `docker.options`
The build tool `docker` allow you to define an `options` section for the following settings:
- [`target`](/docs/cli/image-building/configuration/build-options#target) defining the build target for multi-stage builds
- [`network`](/docs/cli/image-building/configuration/build-options#network) to define which network to use during building (e.g. `docker build --network=host`)
- [`buildArgs`](/docs/cli/image-building/configuration/build-options#buildargs) to pass arguments to the Dockerfile during the build process

See [Build Options](/docs/cli/image-building/configuration/build-options) for details.



## `kaniko`
Using `kaniko` as build tool allows you to build images direclty inside your Kubernetes cluster without a Docker daemon. DevSpace simply starts a build pod and builds the image using `kaniko`.

> After the build process completes, the build pod started for the kaniko build process will be deleted again.

To set `kaniko` as default build tool use the following configuration:
```yaml
images:
  backend:
    image: john/appbackend
    build:
      kaniko:
        cache: true
```

### `kaniko.cache`
The `cache` option expects a boolean that states if kaniko should make use of layer caching by pulling the previously build image and using the layers of this image as cache.

#### Default Value For `cache`
```yaml
cache: true
```

#### Example: Building Images with kaniko
```yaml
images:
  backend:
    image: john/appbackend
    build:
      kaniko:
        cache: true
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    build:
      kaniko:
        cache: false
```
**Explanation:**  
- The first image `backend` would be built using kaniko and make use of the build cache.
- The second image `frontend` would be built using kaniko and **not** use the build cache.


### `kaniko.snapshotMode`
The `snapshotMode` option expects a string that can have the following values:
- `full` tells kaniko to do a full filesystem snapshot (default)
- `time` tells kaniko to do a filesystem snapshot based on `mtime`

> See [limitations related to kaniko snapshots using `mtime`](https://github.com/GoogleContainerTools/kaniko#mtime-and-snapshotting).

#### Default Value For `snapshotMode`
```yaml
snapshotMode: full
```

#### Example: Building Images with kaniko
```yaml
images:
  backend:
    image: john/appbackend
    build:
      kaniko:
        cache: true
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    build:
      kaniko:
        snapshotMode: time
```
**Explanation:**  
- The first image `backend` would be built using kaniko and creating full filesystem snapshots.
- The second image `frontend` would be built using kaniko and calculate filesystem snapthots only based on `mtime`.


### `kaniko.flags`
The `flags` option expects an array of strings that will be passed as flags and values for these flags when running the kaniko build command.

> Take a look at the kaniko documentation for a full [list of available flags](https://github.com/GoogleContainerTools/kaniko#additional-flags).

#### Default Value For `flags`
```yaml
flags: []
```

#### Example: Passing Flags to kaniko
```yaml
images:
  backend:
    image: john/appbackend
    build:
      kaniko:
        flags:
        - --cache-dir
        - /tmp
        - --verbosity
        - debug
```
**Explanation:**  
The image `backend` would be built using kaniko and the flags `--cache-dir=/tmp --verbosity=debug` would be set when running the build command within the kaniko pod used for image building.


### `kaniko.namespace`
The `namespace` option expects a string stating a namespace that should be used to start the kaniko build pod in.

#### Default Value For `flags`
```yaml
namespace: "" # defaults to the default namespace of the current kube-context
```

#### Example: Different Namespace For kaniko
```yaml
images:
  backend:
    image: john/appbackend
    build:
      kaniko:
        namespace: build-namespace
```
**Explanation:**  
The image `backend` would be built using kaniko and the build pod started to run the kaniko build process would be created within the namespace `build-namespace` within the cluster that the current kube-context points to.


### `kaniko.insecure`
The `insecure` option expects a boolean stating if kaniko should allow to push to an insecure (plain HTTP instead of HTTPS) registry.

> This option should only be set to `true` for testing purposes.

#### Default Value For `insecure`
```yaml
insecure: false
```

#### Example: Push to Insecure Registry With kaniko
```yaml
images:
  backend:
    image: 123.456.789.0:5000/john/appbackend
    build:
      kaniko:
        namespace: build-namespace
```
**Explanation:**  
The image `backend` would be built using kaniko and pushing to the insecure registry `123.456.789.0:5000` would be allowed.


### `kaniko.pullSecret`
The `pullSecret` option expects a string with the name of a Kubernetes secret which is used by kaniko as pull secret (e.g. for pulling the base image defined in the `FROM` statement of the Dockerfile).

> In most cases, DevSpace already makes sure that kaniko gets the correct pull secrets to push and pull to registries.

#### Default Value For `pullSecret`
```yaml
pullSecret: ""
```

#### Example: Pull Secret For kaniko
```yaml
images:
  backend:
    image: john/appbackend
    build:
      kaniko:
        pullSecret: custom-pull-secret
```
**Explanation:**  
The image `backend` would be built using kaniko and kaniko would use the Kubernete secret `custom-pull-secret` to pull images from registries that require authentication.

### `kaniko.options`
The build tool `kaniko` allow you to define an `options` section for the following settings:
- [`target`](/docs/cli/image-building/configuration/build-options#target) defining the build target for multi-stage builds
- [`network`](/docs/cli/image-building/configuration/build-options#network) to define which network to use during building (similar to `docker build --network=host`)
- [`buildArgs`](/docs/cli/image-building/configuration/build-options#buildargs) to pass arguments to the Dockerfile during the build process

See [Build Options](/docs/cli/image-building/configuration/build-options) for details.



## `custom`
Using `custom` as build tool allows you to define a custom command for building images. This is particularly useful if you want to use a remote build system such as Google Cloud Build.

> Make sure your `custom` build command terminates with exit code 0 when the build process was successful.

### `custom.command`
The `onChange` option expects an array of strings which represent paths to files or folders that should be watched for changes. DevSpace uses these paths and the hash values it stores about these paths for evaluating if an image should be rebuilt or if the image building can be skipped because the context of the image has not changed.

> It is highly recommended to specify this option when using `custom` as build tool in order to speed up the build process.

#### Example: Building Images With `custom` Build Command
```yaml
images:
  backend:
    image: john/appbackend
    build:
      custom:
        command: ./build
        args:
        - "--arg1"
        - "arg-value-1"
        - "--arg2"
        - "arg-value-2"
```
**Explanation:**  
The image `backend` would be built using the command `./build --arg1=arg-value-1 --arg2=arg-value-2 "[IMAGE]:[TAG]"` while `[IMAGE]` would be replaced with the `image` option (in this case: `john/appbackend`) and `[TAG]` would be replaced with the tag generated according to the [tagging schema](/docs/cli/image-building/configuration/overview-specification#images-tag-tagging-schema).


### `custom.args`
The `onChange` option expects an array of strings which represent additional flags and arguments that should be passed to the custom build command.

#### Default Value For `args`
```yaml
onChange: []
```

#### Example
**See "[Example: Building Images With `custom` Build Command](#example-building-images-with-custom-build-command)"**


### `custom.imageFlag`
The `onChange` option expects a string which states the name of the flag that is used to pass the image name (including auto-generated tag) to the custom build script defined in `command`.

#### Default Value For `imageFlag`
```yaml
imageFlag: "" # Defaults to passing image and tag as an argument instead of using a flag
```

#### Example: Defining `imageFlag` For `custom` Build Command
```yaml
images:
  backend:
    image: john/appbackend
    build:
      custom:
        command: ./build
        imageFlag: image
        args:
        - "--arg1"
        - "arg-value-1"
        - "--arg2"
        - "arg-value-2"
```
**Explanation:**  
The image `backend` would be built using the command `./build --arg1=arg-value-1 --arg2=arg-value-2 --image="[IMAGE]:[TAG]"` while `[IMAGE]` would be replaced with the `image` option (in this case: `john/appbackend`) and `[TAG]` would be replaced with the tag generated according to the [tagging schema](/docs/cli/image-building/configuration/overview-specification#images-tag-tagging-schema).


### `custom.onChange`
The `onChange` option expects an array of strings which represent paths to files or folders that should be watched for changes. DevSpace uses these paths and the hash values it stores about these paths for evaluating if an image should be rebuilt or if the image building can be skipped because the context of the image has not changed.

> It is highly recommended to specify this option when using `custom` as build tool in order to speed up the build process.

#### Default Value For `onChange`
```yaml
onChange: []
```

#### Example: OnChange Option For custom Build Command
```yaml
images:
  backend:
    image: john/appbackend
    build:
      custom:
        command: ./build
        imageFlag: image
        onChange:
        - some/path/
        - another/path/file.txt
```
**Explanation:**  
The image `backend` would be built using the command `./build --image="[IMAGE]:[TAG]"` and DevSpace would skip the build if none of the files within `some/path/` nor the file `another/path/file.txt` has changed since the last build.



## `disabled`
The `disabled` option expects a boolean and allows you to disable image building for an image.

This config option may be useful when developing an application that takes long to build and does not need to be rebuild very frequently because the [file synchronization](/docs/cli/development/configuration/file-synchronization) in development mode is much quicker to update the development container than rebuilding the image. In this case, you could set `disabled: true` and manually rebuild if needed using `devspace build`.

#### Default Value For `disabled`
```yaml
disabled: false
```

#### Example: Disabling Image Building
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    build:
      disabled: true
```
**Explanation:**  
- The first image `backend` would be built using [`docker`](#docker-default) which is the default build tool.
- The second image `frontend` would not be built at all.
