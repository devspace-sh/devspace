---
title: Building Images with Docker
sidebar_label: docker
---

By default, DevSpace CLI builds your images with a local Docker daemon if Docker is installed and running. The DevSpace configuration provides a lot of options for customizing image building with Docker. The following config snippet shows some of the available options. For details, refer to the [Full Config Reference](/docs/configuration/reference#images-builddocker)

```yaml
images:
  default:
    image: dscr.io/username/image
    build:
      docker:
        preferMinikube: true
        disableFallback: false
        options:
          buildArgs:
            someArg: argValue
            anotherArg: anotherValue
```

The above config shows a couple of common options:
- If you are using minikube to deploy your application to, DevSpace CLI uses the Docker daemon inside the minikube VM instead of the Docker daemon on your host machine. If you wish to always build images with your host machine's Docker daemon, set `preferMinikube: false`.
- By default, DevSpace CLI uses `kaniko` as a fallback build tool when Docker is not running. You can disable this behavior by setting `disableFallback: false`.
- DevSpace CLI can pass certain configurations directly to the Docker daemon for building an image. The most commonly used is `buildArgs`. Additionally, DevSpace CLI allows to specify a `target` and a `network` flag for Docker builds.
