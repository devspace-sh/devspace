---
title: Image
---

Components deploy pods which are a set of containers. These containers are created based on Docker images. To define the image for a container, simply set the `image` value for the container:
```yaml
components:
- name: my-backend
  containers:
  - image: dscr.io/username/my-backend-image
  - image: nginx:1.15
```
The example above would create a pod with two containers:
1. The first container would be create from the image `dscr.io/username/my-backend-image`
2. The second container would be created from the `nginx` image on [Docker Hub](https://hub.docker.com) which is tagged as version `1.15`

> If you are using a private Docker registry, make sure to [login to this registry](/docs/image-building/authentication).
