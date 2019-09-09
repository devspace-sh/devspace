---
title: Workflow & Basics
---

DevSpace fully automates the manual work of building, tagging and pushing Docker images.

<br>
<img src="/img/processes/image-building-process-devspace.svg" alt="DevSpace Image Building Process" style="width: 100%;">

## Commands Triggering Image Building
When you run one of the following commands, DevSpace will run the image building process:
- `devspace build` (only image building without deployment)
- `devspace deploy` (before deploying the application)
- `devspace dev` (before deploying the application and starting the development mode)

## Image Building Process
DevSpace loads the `images` configuration from `devspace.yaml` and builds all images in parallel. The multi-threded, parallel build process of DevSpace speeds up image building drastically, especially when building many images and using remote build methods. 

> You can use the `--build-sequential` flag to tell DevSpace to build images sequentially instead of using the parallel approach.


### 1. Load Dockerfile
DevSpace loads the contents of the Dockerfile specified in `dockerfile` (defaults to `./Dockerfile`). 

> Dockerfile paths used in `dockerfile` should be relative to the `devspace.yaml`.

### 2. Apply Entrypoint Override (if configured) 
DevSpace allows you to apply an in-memory override of a Dockerfile's `ENTRYPOINT` by configuring the `entrypoint` option for the image. Similar to the Dockerfile `ENTRYPOINT`, the `entrypoint` option should be defined as an array. 

> Configuring `ENTRYPOINT` overrides can be particularly useful when defining different [config profiles](#TODO) in your `devspace.yaml`.

### 3. Load Build Context
DevSpace loads the the context to build this image as specified in `context` (defaults to `./`). The context path serves as root directory for Dockerfile statements like `ADD` or `COPY`. 

See: [What does "context" mean in terms of image building?](#what-does-context-mean-in-terms-of-image-building) 

> Context paths used in `context` should be relative to the `devspace.yaml`.

### 4. Skip Image (if possible)
DevSpace tries to speed up image building by skipping images when they have not changed since the last build process. To do this, DevSpace caches the following information after building an image:
- a hash of the `Dockerfile` used to build the image (including ENTRYPOINT override)
- a hash over all files in the `context` used to build this image (excluding files in `.dockerignore`)

Next time you trigger the image building process, DevSpace will generate these hashes again and compare them to the hashes of the last image building process. If all newly generated hashes are equal to the old ones stored during the last image building process, then nothing has changed and DevSpace will skip this image.

> You can use the `-b / --force-build` flag to tell DevSpace to build all images even if nothing has changed.

### 5. Build Image
DevSpace uses one of the following [build tools](/docs/cli/image-building/build-tools/what-are-build-tools) to create an image based on your Dockerfile and the provided context:
- [`docker`](/docs/cli/image-building/build-tools/docker) for building images using a Docker daemon (default, [prefers Docker daemon of local Kubernetes clusters](#docker-daemon-of-local-kubernetes-clusters))
- [`kaniko`](/docs/cli/image-building/build-tools/kaniko) for building images directly inside Kubernetes ([fallback for `docker`](#kaniko-as-fallback-for-docker))
- [`custom`](/docs/cli/image-building/build-tools/custom-build-commands) for building images with a custom build command (e.g. for using Google Cloud Build)

<details>
<summary>
#### Kaniko as Fallback for Docker
</summary>

When using `docker` as build tool, DevSpace checks if Docker is installed and running. If Docker is not installed or not running, DevSpace will use kaniko as fallback to build the image.

</details>

<details>
<summary>
#### Docker Daemon of Local Kubernetes Clusters
</summary>

DevSpace preferably uses the Docker daemon running in the virtual machine that belongs to your local Kubernetes cluster instead of your regular Docker daemon. This has the advantage that images do not need to be pushed to a registry because Kubernetes can simply use the images available in the Docker daemon belonging to the kubelet of the local cluster. Using this method is only possible when your current kube-context points to a local Kubernetes cluster and is named `minikube`, `docker-desktop` or `docker-for-desktop`.

</details>

### 6. Tag Image
DevSpace automatically tags all images after building them using a tagging schema that you can customize using the `tag` option. By default, this option is configured to generate a random string consisting of 5 characters. 

[Learn more about defining a custom tagging schema](#TODO)

> Before deploying your application, DevSpace will use the newly generated image tags and replace them in-memory in the values for your [Helm charts](/docs/cli/deployment/helm-charts/configuration/overview-specification) and [components](/docs/cli/deployment/components/configuration/overview-specification), so they will be deployed using the most recently built images.

### 7. Push Image
DevSpace automatically pushes your images to the respective registry that should be specified as part of the `image` option. Just as with regular Docker images, DevSpace uses Docker Hub if no registry hostname is provided within `image`.

> You can skip this step when proving the `--skip-push` flag. Beware that deploying your application after using `--skip-push` will only work when [using a local Kubernetes cluster](#skip-image-push-for-local-clusters-if-possible).

<details>
<summary>
#### Registry Authentication
</summary>

DevSpace uses the same credential store as Docker. So, if you already have Docker installed and you are logged in to a private registry before, DevSpace will be able to push to this registry. So, if you want to push to a registry using DevSpace, simply authenticate using this command:
```bash
docker login            # for Docker Hub
docker login [REGISTRY] # for any other registry (e.g. my-registry.tld)
```

> If you do not have Docker installed, DevSpace initializes a Docker credential store for you and store your login information just like Docker would do it (currently only for Docker Hub or dscr.io).

</details>

<details>
<summary>
#### Skip Image Push for Local Clusters (if possible)
</summary>

If you are using a local Kubernetes cluster, DevSpace will try to [build the image using the Docker deamon of this local cluster](#docker-daemon-of-local-kubernetes-clusters). If this process is successful, DevSpace will skip the step of pushing the image to a registry as it is not required for deploying your application.

</details>


### 8. Create Image Pull Secret
When deploying your application via DevSpace, Kubernetes needs to be able to pull your images from the registry that is used to store your images when pushing them. For this purpose, Kubernetes relies on so-called image pull secrets. DevSpace can automatically create these secrets for you, if you configure `createPullSecret: true` for the respective image in your `devspace.yaml`.

> You do not need to change anything in your Kubernetes manifests, helm charts or components to use the image pull secrets that DevSpace creates because DevSpace automatically [adds the secrets to the service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account) used to deploy your project.


## Best Practices

### Optimize Dockerfiles
#TODO

### Use `.dockerignore`
DevSpace respects the `.dockerignore` file when defined on the root level of your context directory. This file follows a similar syntax as the `.gitignore` file but instead of excluding files from git, the `.dockerignore` file defines files and folders which should not be included in the context for building an image. 

> Adding paths to the `.dockerignore` file makes sure that DevSpace is not forced to rebuild images when files belonging to theses paths change.

It can often be useful to:
- Add `devspace.yaml` to `.dockerignore` to prevent config changes from triggering image rebuilding (`devspace init` does this by default)
- Add temporary files (e.g. `.DS_Store`) to `.dockerignore` (DevSpace ALWAYS ignores `.devspace/` temp folder even if not specified in `.dockerignore`)
- Add dependency folders to `.dockerignore`, here are a few examples of dependency folders for different languages:

#### Recommended Paths for `.dockerignore`
| Language / Dependency Tool | `.dockerignore` statements                                                                                    |
| ------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| All Languages | `devspace.yaml` |
| PHP / composer | `composer.phar`<br>`vendor/` |
| Node.js / npm | `node_modules/`<br>`npm-debug.log*`<br>`report.[0-9]*.[0-9]*.[0-9]*.[0-9]*.json`<br>`pids`<br>`*.pid*`<br>`*.seed*`<br>`*.pid.lock*` |
| Python / pip | `__pycache__/`<br>`wheels/`<br>`pip-log.txt`<br>`pip-wheel-metadata/` |


<br>

---
## FAQ

<details>
<summary>
### What does "context" mean in terms of image building?
</summary>
The context is archived and sent to the Docker daemon before starting to process the Dockerfile. All references of local files within the Dockerfile are relative to the root directory of the context. 

That means that a Dockerfile statement such as `COPY ./src /app` would copy the folder `src/` within the context path into the path `/app` within the container image. So, if the context would be `/my/project/database`, for example, the folder that would be copied into `/app` would have the absolute path `/my/project/database/src` on your local computer.

> Paths to Dockerfiles and image contexts are always relative to the root directory of your project (i.e. the folder where your `.devspace/` folder is inside).
</details>
