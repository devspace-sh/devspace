---
title: Configure Docker images
---

DevSpace.cli allows you to build, tag and push all the images that you need for deploying your application.

## Default image created by `devspace init`
When running `devspace init` within your project, DevSpace.cli defines an image called `default` within your config file `.devspace/config.yaml`.
```yaml
images:
  default:
    image: dscr.io/username/devspace
```
Because this image called `default` only has the `image` option configured, DevSpace.cli will automatically conclude that:

1. The image should be built using your local Docker daemon
2. The Dockerfile for building the image will be located inside the root folder of your project (i.e. ./Dockerfile)
2. The context for building the image will be the root folder of your project (i.e. ./)

## Add additonal images
To tell DevSpace.cli to build an additional image, simply use the `devspace add image` command.
```bash
devspace add image database --dockerfile=./db/Dockerfile --context=./db --image=dscr.io/username/mysql
```

The command shown above would add a new image to your DevSpace configuration. The resulting configuration would look similar to this one:

```yaml
images:
  database:                         # from --name
    image: dscr.io/username/image   # from args[0]
    build:
      dockerfile: ./db/Dockerfile   # from --dockerfile
      context: ./db                 # from --context
```

## Remove an image
Instead of manually removing an image from your configuration file, you can also use the `devspace remove image` command.
```bash
devspace remove image database
```
The command shown above would remove the image with name `database` from your DevSpace configuration.

## Build images with kaniko (experimental)
Instead of using your local Docker daemon to build your images, you can also use [kaniko](https://github.com/GoogleContainerTools/kaniko) to build Docker images. Using kaniko has the advantage that you are building the image inside a container that runs remotely on top of Kubernetes. Using DevSpace.cloud, this container would run inside the Space that you are currently working with.
```yaml
images:
  default:
    image: dscr.io/username/devspace
    build:
      kaniko:
        cache: true
```
The config excerpt shown above would tell DevSpace.cli to build the image `default` with kaniko and to use caching while building the image.

> In comparison to using a local Docker daemon, **kaniko is currently very slow** at building images. Therefore, it is currently recommended to use Docker for building images.

## Skip image pushing (for development with minikube)
If you are using minikube for development, you usually do not need to push your images to a registry because DevSpace.cli will build your images with minikube's Docker daemon and the image will already be present and does not need to be pulled from a registry.
```yaml
images:
  default:
    image: my-registry.tld/username/image
    skipPush: true
```
Defining `skipPush: true` tells DevSpace.cli not to push an image after building and tagging it.


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
