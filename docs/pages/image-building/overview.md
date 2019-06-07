---
title: Overview & Basics
---

DevSpace CLI allows you to build, tag and push all the images that you need for deploying your application.

> If you have multiple Dockerfiles in your project (e.g. in case of a monorepo), you can also tell DevSpace CLI to build multiple images in a row by [adding new images to `devspace.yaml`](/docs/image-building/add-images).

## Image building process
DevSpace CLI fully automates the manual work of building, tagging and pushing Docker images and executes the following steps during `devspace deploy` and `devspace dev`:
1. Build a new image (if the Dockerfile or the Docker context has changed)
2. Apply [entrypoint overrides](/docs/development/overrides) for development (only when running `devspace dev`)
3. Tag this new image with an auto-generated tag
4. Push this image to any [Docker registry](/docs/image-building/registries/authentication) of your choice
5. Create [image pull secrets](/docs/image-building/registries/pull-secrets) for your registries

### Replacing image tags before deployment
After building your images as part of `devspace deploy` or `devspace dev`, DevSpace CLI will continue with deploying your application as defined in the `deployments`. Before deploying, DevSpace CLI will use the newly generated tag and replace every occurence of the same image in your deployment files (e.g. Helm charts or Kubernetes manifests) with the newly generated tag, so that you are always deploying the newest version of your application. This tag replacement happens entirely in-memory, so your deployment files will not be altered.

### Skipping image building
DevSpace CLI automatically skips image building when neither the Dockerfile nor the context has changed since the last time an image bas been build from the repective Dockerfile.

## Configuring the image building process
There are a couple of configuration options to influence the image building process.

<details>
<summary>
### Creating image pull secrets
</summary>
To make sure that Kubernetes can pull your image even when you are pushing to a private registry (such as dscr.io), DevSpace CLI will also create an [image pull secret](/docs/image-building/registries/pull-secrets) containing credentials for your registry.

## Default image created by `devspace init`
When running `devspace init` within your project, DevSpace CLI defines an image called `default` within your config file `devspace.yaml`.
```yaml
images:
  default:
    image: dscr.io/username/devspace
```
Because this image called `default` only has the `image` option configured, DevSpace CLI will automatically conclude that:

1. The image should be built using your local Docker daemon
2. The Dockerfile for building the image will be located inside the root folder of your project (i.e. ./Dockerfile)
3. The context for building the image will be the root folder of your project (i.e. ./)
</details>

<details>
<summary>
### Build images with kaniko instead of Docker (experimental)
</summary>
Instead of using your local Docker daemon to build your images, you can also use [kaniko](https://github.com/GoogleContainerTools/kaniko) to build Docker images. Using kaniko has the advantage that you are building the image inside a container that runs remotely on top of Kubernetes. Using DevSpace Cloud, this container would run inside the Space that you are currently working with.
```yaml
images:
  default:
    image: dscr.io/username/devspace
    build:
      kaniko:
        cache: true
```
The config excerpt shown above would tell DevSpace CLI to build the image `default` with kaniko and to use caching while building the image.

> In comparison to using a local Docker daemon, **kaniko is currently rather slow** at building images. Therefore, it is currently recommended to use Docker for building images.
</details>

<details>
<summary>
### Skip image pushing (for development with minikube)
</summary>
If you are using minikube for development, you usually do not need to push your images to a registry because DevSpace CLI will build your images with minikube's Docker daemon and the image will already be present and does not need to be pulled from a registry.
```yaml
images:
  default:
    image: my-registry.tld/username/image
    build:
      docker:
        skipPush: true
```
Defining `skipPush: true` tells DevSpace CLI not to push an image after building and tagging it.
</details>


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
