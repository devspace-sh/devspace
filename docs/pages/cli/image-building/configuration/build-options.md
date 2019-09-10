---
title: Configuring Build Options
sidebar_label: Build Options
---

The build tools `docker` and `kaniko` allow you to define an `options` section for the following settings:
- `target` defining the build target for multi-stage builds
- `network` to define which network to use during building (e.g. `docker build --network=host`)
- `buildArgs` to pass arguments to the Dockerfile during the build process


## `target`
The `target` option expects a string with state the build target when using multi-stage builds.

#### Example: Defining a Build Target for Docker
```yaml
images:
  backend:
    image: john/appbackend
    build:
      docker:
        options:
          target: production
```
**Explanation:**  
The image `backend` would be built using `docker` and the target `production` would be used for building the image as defined in the `Dockerfile`.


## `network`
The `network` option expects a string with state the network setting for building the image.

#### Example: Defining a Network for Docker
```yaml
images:
  backend:
    image: john/appbackend
    build:
      docker:
        options:
          network: host
```
**Explanation:**  
The image `backend` would be built using `docker` and `docker build` would be called using the `--network=host` flag.


## `buildArgs`
The `buildArgs` option expects a map of buildArgs representing values for the `--build-arg` flag used for `docker` or `kaniko` build commands.

#### Example: Defining Build Args for Docker
```yaml
images:
  backend:
    image: john/appbackend
    build:
      docker:
        options:
          buildArgs:
            arg1: arg-value-2
            arg2: arg-value-2
```
**Explanation:**  
The image `backend` would be built using `docker` and `docker build` would be called using the `--build-arg arg1=arg-value-1 --build-arg arg2=arg-value-2` flags.
