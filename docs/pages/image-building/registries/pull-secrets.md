---
title: Pull secrets
---

When you push images to a private registry, you need to login to this registry beforehand (e.g. using `docker login`). When Kubernetes tries to pull images from a private registry, it also has to provide credentials to be authorized to pull images from this registry. The way to tell Kubernetes these credentials is to create a Kubernetes secret with these credentials. Such a secret is called image pull secret.

> DevSpace CLI can automatically create image pull secrets and add them to the `default` service account for images within your DevSpace configuration. You can enable this via the `createPullSecret` option in an image configuration.

Example:
```yaml
images:
  default:
    image: dscr.io/myusername/devspace
    # This tells DevSpace to create an image pull secret and add it to the default service account during devspace deploy & devspace dev
    createPullSecret: true
```

## Creating pull secrets manually
If you want to create your pull secret manually you can do this via the following command:

```bash
kubectl create secret docker-registry my-pull-secret --docker-server=[REGISTRY_URL] --docker-username=[REGISTRY_USERNAME] --docker-password=[REGISTRY_PASSWORD] --docker-email=[YOUR_EMAIL]
```

This `kubectl` command would create an image pull secret called `my-pull-secret`. 

[Learn more about image pull secrets in Kubernetes.](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)

However you also have to add it to your service account by running this command:
```bash
kubectl edit serviceaccount default
```

Then add the just created pull secret:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
  namespace: default
secrets:
- name: default-token-6k6fc
imagePullSecrets:
- name: my-pull-secret
```

Save and now you should be able to pull images from that registry.
