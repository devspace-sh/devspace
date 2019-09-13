---
title: Image
id: version-v3.5.18-image
original_id: image
---

Components deploy pods which are a set of containers. These containers are created based on Docker images. To define the image for a container, simply set the `image` value for the container:
```yaml
deployments:
- name: my-backend
  component:
    containers:
    - image: dscr.io/username/my-backend-image
    - image: nginx:1.15
```
The example above would create a pod with two containers:
1. The first container would be create from the image `dscr.io/username/my-backend-image`
2. The second container would be created from the `nginx` image on [Docker Hub](https://hub.docker.com) which is tagged as version `1.15`

> If you are using a private Docker registry, make sure to [logged into this registry](/docs/image-building/registries/authentication).
