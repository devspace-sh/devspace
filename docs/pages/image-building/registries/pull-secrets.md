---
title: Pull secrets
---

When you push images to a private registry, you need to login to this registry beforehand (e.g. using `docker login`). When Kubernetes tries to pull images from a private registry, it also has to provide credentials to be authorized to pull images from this registry. The way to tell Kubernetes these credentials is to create a Kubernetes secret with these credentials. Such a secret is called image pull secret.

> By default, DevSpace CLI automatically creates and manages image pull secrets for all the images within your DevSpace configuration.

## Creating your own secrets
If you are not using dscr.io, it is recommended to create your own image pull secrets.

```bash
devspace use space [SPACE_NAME]
kubectl create secret docker-registry my-pull-secret --docker-server=[REGISTRY_URL] --docker-username=[REGISTRY_USERNAME] --docker-password=[REGISTRY_PASSWORD] --docker-email=[YOUR_EMAIL]
```

This `kubectl` command would create an image pull secret called `my-pull-secret`. 

[Learn more about image pull secrets in Kubernetes.](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)

To tell DevSpace CLI not to create an additional secret, you can use the config option `createPullSecret` and set it to `false` for the respective image.

```yaml
images:
  default:
    image: my-registry.tld/username/image
    createPullSecret: false
```

To use your custom image pull secret, the DevSpace Helm Chart provides an array called `pullSecrets` within `chart/values.yaml`.

```yaml
pullSecrets:
- my-pull-secret
```

Adding your pull secret `my-pull-secret` to this array will allow you to pull images with this secret and use the image within your deployment configuration (i.e. in `components[*].containers[*].image`).
