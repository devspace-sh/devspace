---
title: Configure Image Building
sidebar_label: Configuration
id: version-v4.0.1-overview-specification
original_id: overview-specification
---

Images are configured within the `images` section of the `devspace.yaml`.
```yaml
images:                 # DevSpace will build these images in parallel and push them to the respective registries
  {image-a}: ...        # tells DevSpace how to build image-a
  {image-b}: ...        # tells DevSpace how to build image-b
  {image-c}: ...        # tells DevSpace how to build image-c
  ... 
```

> To speed up the build process, the images you specify under `images` will all be built in parallel (unless you use the `--build-sequential` flag).

## Image Definition
The `images` section in `devspace.yaml` is map with keys representing the name of the image and values representing the image definition including `tag`, `dockerfile` etc.
```yaml
images:                             # map[string]struct | Images to be built and pushed
  image1:                           # string   | Name of the image
    image: dscr.io/username/image   # string   | Image repository and name 
    tag: v0.0.1                     # string   | Tagging schema
    dockerfile: ./Dockerfile        # string   | Relative path to the Dockerfile used for building (Default: ./Dockerfile)
    context: ./                     # string   | Relative path to the context used for building (Default: ./)
    createPullSecret: true          # bool     | Create a pull secret containing your Docker credentials (Default: false)
    build: ...                      # struct   | Build options for this image
  image2: ...
```


### `images[*].image`
The `image` option expects a string containing the image repository including registry and image name. 

- Make sure you [authenticate with the image registry](/docs/cli/image-building/workflow-basics#registry-authentication) before using in here.
- For Docker Hub images, do not specify a registry hostname and use just the image name instead (e.g. `mysql`, `my-docker-username/image`).

#### Example: Multiple Images
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
```
**Explanation:**
- The first image `backend` would be tagged as `appbackend:[TAG]` pushed to Docker Hub using the path `john` (which generally could be your Docker Hub username).
- The second image `frontend` would be tagged as `appfrontend:[TAG]` and pushed to `dscr.io` using the path `${DEVSPACE_USERNAME}` which is a [dynamic config variable](/docs/cli/configuration/variables) that resolves to your username in DevSpace Cloud. 

> See **[`images[*].tag` *Tagging Schema*](#images-tag-tagging-schema)** for details on how the image `[TAG]` would be set in this case.


### `images[*].tag` *Tagging Schema*
The `tag` option expects a string containing a custom tagging schema used to automatically tag images before pushing them to the registry. The tagging schema can contain [dynamic config variables](/docs/cli/configuration/variables). While you can define your own config variables, DevSpace provides a set of pre-defined variables. The most commonly used variables for tagging are:
- **DEVSPACE_RANDOM**: A random 6 character long string
- **DEVSPACE_TIMESTAMP** A unix timestamp when the config was loaded
- **DEVSPACE_GIT_COMMIT**: A short hash of the local repos current git commit
- **DEVSPACE_USERNAME**: The username currently logged into devspace cloud

**See also: [How does DevSpace replace tags in my deployments?](/docs/cli/deployment/workflow-basics#3-tag-replacement)**

> **Make sure tags are unique** when defining a custom tagging schema. Unique tags ensure that when your application gets started with the newly built image instead of using an older, cached version. 
> 
> Therefore, it is highly recommended to either use `DEVSPACE_RANDOM` or `DEVSPACE_TIMESTAMP` as a suffix in your tagging schema (see [example below](#example-custom-tagging-schema)).

#### Default Value For `tag`
```yaml
tag: ${DEVSPACE_RANDOM}
```

#### Example: Custom Tagging Schema
```yaml
images:
  backend:
    image: john/appbackend
    tag: dev-${DEVSPACE_USERNAME}-backend-${DEVSPACE_GIT_COMMIT}-${DEVSPACE_RANDOM}
```
**Explanation:**  
The above example would generate tags such as `dev-john-backend-b6caf8a-Jak9i` which would result from concatenating the following substrings:
- `dev-` static string 
- `john` [DevSpace Cloud](/docs/cloud/what-is-devspace-cloud) username
- `-backend-` static string 
- `b6caf8a` latest git commit hash on current local branch
- `-` static string
- `Jak9i` auto-generated random string


### `images[*].dockerfile`
The `dockerfile` option expects a string with a path to a `Dockerfile`.
- The path in `dockerfile` should be relative to the `devspace.yaml`.
- When setting the `dockerfile` option it is often useful to set the [`context` option](#images-context) as well.
- To share your configuration with team mates, make sure `devspace.yaml` and all `Dockerfiles` used in the `images` section are checked into your code repository.

#### Default Value For `dockerfile`
```yaml
dockerfile: ./Dockerfile
```

#### Example: Different Dockerfiles
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    dockerfile: frontend/Dockerfile
    context: frontend/
```
**Explanation:**  
- The first image would be built using the Dockerfile in `./Dockerfile` and the context path `./`.
- The second image would be built using the Dockerfile in `./frontend/Dockerfile` and the context path `./frontend/`.
- Switching the `context` for image `frontend` would assure that a statement like `ADD file.txt` or  `COPY file.txt .` in `./frontend/Dockerfile` would use the local file `./frontend/file.txt` instead of `./file.txt`.
- In this example, it would probably be useful to have a `./.dockerignore` file which ignores the `frontend/` folder when building the first image called `backend`.

> See **[Best Practices for Image Building](/docs/cli/image-building/advanced-topics/best-practices)** for details on how to optimize your Dockerfiles and use `.dockerignore` for faster image building.

### `images[*].context`
The `context` option expects a string with a path to the folder used as context path when building the image. The context path serves as root directory for Dockerfile statements like ADD or COPY.a

**See: [What does "context" mean in terms of image building?](/docs/cli/image-building/workflow-basics#what-does-context-mean-in-terms-of-image-building)**

#### Default Value For `context`
```yaml
context: ./
```

#### Example
**See "[Example: Different Dockerfiles](#example-different-dockerfiles)"**


## Overriding `ENTRYPOINT` &amp; `CMD`

### `images[*].entrypoint`
The `entrypoint` option expects an array of strings which tells DevSpace to overrides the `ENTRYPOINT` defined in the `Dockerfile` during the image building process.

[Learn more about how overrides are applied during image building](/docs/cli/image-building/workflow-basics#2-apply-entrypoint-override-if-configured).

> Overriding `ENTRYPOINT` also works for multi-stage builds.

> If you are overriding the `ENTRYPOINT`, it is often useful to also [override the `CMD` statement](#images-cmd). If you want to define `entrypoint: ...` for an image and you do **not** want the `CMD` statement from the Dockerfile, add `cmd: []` to the image configuration in your `devspace.yaml`.

> If you are overriding the Dockerfile `ENTRYPOINT` using the `entrypoint` option, it only affects the image but **not** the deployment. If a deployment using this image defines the [`command` option](/docs/cli/deployment/components/configuration/containers#command), it will take precedence over the Dockerfile `ENTRYPOINT` as well as over the `entrypoint` settings configured in `devspace.yaml`.

#### Default Value For `entrypoint`
```yaml
entrypoint: []
```

#### Example: Override ENTRYPOINT For Image
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    entrypoint:
    - npm
    - run
    - dev
```
**Explanation:**  
- The first image `backend` will be built using the regular `ENTRYPOINT` (e.g. `[npm, start]`) defined by the Dockerfile located in `./Dockerfile`
- The second image `frontend` will be built using the same Dockerfile but instead of the original `ENTRYPOINT`, DevSpace would use the `[npm, run, dev]` as value for `ENTRYPOINT`


### `images[*].cmd`
The `cmd` option expects an array of strings which tells DevSpace to overrides the `CMD` defined in the `Dockerfile` during the image building process.

[Learn more about how overrides are applied during image building](/docs/cli/image-building/workflow-basics#2-apply-entrypoint-override-if-configured).

> Overriding `CMD` also works for multi-stage builds.

> `CMD` generally defines the arguments for `ENTRYPOINT`.

> If you are overriding the Dockerfile `CMD` using the `cmd` option, it only affects the image but **not** the deployment. If a deployment using this image defines either the [`command` option](/docs/cli/deployment/components/configuration/containers#command) or the [`args` option](/docs/cli/deployment/components/configuration/containers#args), it will take precedence over the Dockerfile `CMD` as well as over the `cmd` settings configured in `devspace.yaml`.

#### Default Value For `cmd`
```yaml
cmd: []
```

#### Example: Override CMD For Image
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    cmd:
    - run
    - dev
```
**Explanation:**  
- The first image `backend` will be built using the regular `CMD` (e.g. `[start]`) for `ENTRYPOINT` (e.g. `[npm]`) defined by the Dockerfile located in `./Dockerfile`
- The second image `frontend` will be built using the same Dockerfile but instead of the original `CMD`, DevSpace would use the `[run, dev]` as value for `CMD`


## Image Pull Secrets

### `images[*].createPullSecret`
The `createPullSecret` option expects a boolean that tells DevSpace if a pull secret should be created for the registry where this image will be pushed to.
- If there are multiple images with the **same registry** and any of the images will define `createPullSecret: true`, the pull secret will be created no matter if any of the other images using the same registry explicitly defines `createPullSecret: false`.
- There is **no need to change your Kubernetes manifests, Helm charts or other deployments to reference the pull secret** because DevSpace will [add the pull secret to the service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account) which deploys your project. This ensures that the pull secret is automatically considered by Kubernetes altough it is not explicitly referenced by your pods.
- After running `devspace deploy` or `devspace dev`, you should be able to see the auto-generated pull secrets created by DevSpace when you run the command `kubectl get serviceaccount default -o yaml` and take a look at the `imagePullSecrets` section of this service account.

#### Default Value For `createPullSecret`
```yaml
createPullSecret: false
```

#### Example: Different Dockerfiles
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    createPullSecret: true
```
**Explanation:**  
- The first image `backend` will be pushed to Docker Hub and DevSpace will **not** create a pull secret for Docker Hub because `createPullSecret` defaults to `false`. This makes sense for public images where no login is required to pull the image or where you want to manage the pull secret yourself.
- The second image `frontend` will be pushed to dscr.io and DevSpace will create a pull secret for dscr.io, so Kubernetes is able to pull the image from dscr.io.


## **Build Tools**
The `build` section defines which build tool DevSpace uses to build the image. The following build tools are currently supported:
- [`docker`](/docs/cli/image-building/configuration/build-tools#docker) for building images using a Docker daemon (**default build tool**, [prefers Docker daemon of local Kubernetes clusters](/docs/cli/image-building/workflow-basics#docker-daemon-of-local-kubernetes-clusters))
- [`kaniko`](/docs/cli/image-building/configuration/build-tools#kaniko) for building images directly inside Kubernetes ([fallback for `docker`](/docs/image-building/configuration/build-tools#dockerdisablefallback-kaniko-as-fallback-for-docker))
- [`custom`](/docs/cli/image-building/configuration/build-tools#custom) for building images with a custom build command (e.g. for using Google Cloud Build)
- [`disabled`](/docs/cli/image-building/configuration/build-tools#disabled) for disabling image building for this image

### `images[*].build.docker`
See [Build Tools](/docs/cli/image-building/configuration/build-tools#docker) for details.

### `images[*].build.kaniko`
See [Build Tools](/docs/cli/image-building/configuration/build-tools#kaniko) for details.

### `images[*].build.custom`
See [Build Tools](/docs/cli/image-building/configuration/build-tools#custom) for details.

### `images[*].build.disabled`
See [Build Tools](/docs/cli/image-building/configuration/build-tools#disabled) for details.


## Build Options
The build tools `docker` and `kaniko` allow you to define an `options` section for the following settings:
- `target` defining the build target for multi-stage builds
- `network` to define which network to use during building (e.g. `docker build --network=host`)
- `buildArgs` to pass arguments to the Dockerfile during the build process

### `images[*].build.*.options.target`
See [Build Options](/docs/cli/image-building/configuration/build-options#target) for details.

### `images[*].build.*.options.network`
See [Build Options](/docs/cli/image-building/configuration/build-options#network) for details.

### `images[*].build.*.options.buildArgs`
See [Build Options](/docs/cli/image-building/configuration/build-options#buildargs) for details.

<br>

---
## Useful Commands
DevSpace provides a couple of convenience commands for configuring the `images` section in `devspace.yaml`.

### `devspace add image`
To tell DevSpace to build an additional image, simply use the `devspace add image` command.
```bash
devspace add image database --image=dscr.io/username/mysql --dockerfile=./db/Dockerfile --context=./db
```

This would add a new image called `database` to the `images` section. The resulting configuration would look similar to this one:

```yaml
images:
  database:                         # from args[0]
    image: dscr.io/username/image   # from --image
    dockerfile: ./db/Dockerfile     # from --dockerfile
    context: ./db                   # from --context
```

### `devspace remove image`
Instead of manually removing an image from your configuration file, you can simply run:
```bash
devspace remove image database
```
This command would remove the image `database` from your `devspace.yaml`.
