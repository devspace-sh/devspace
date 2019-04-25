---
title: Use kustomize
---

If you need more than just static manifests, you can use the Kubernetes-native templating solution provided by [kustomize](https://kustomize.io/).

After adding a manifest deployment, the configuration for this deployment will look similar to this example:
```yaml
deployments:
- name: my-deployment
  kubectl:
    manifests:
    - my-manifests/*
    - more-manifests/*
```

You can easily create a `kustomization.yaml` file within your `my-manifests` and within your `more-manifests` folder and tell DevSpace CLI to deploy these manifest via `kustomize` by modifying the configuration as follows:
```yaml
deployments:
- name: my-deployment
  kubectl:
    manifests:
    - my-manifests/
    - more-manifests/
    kustomize: true
```
This configuration would tell DevSpace CLI to deploy our application with the following commands:
```
kubectl apply -k manifests/
kubectl apply -k more-manifests/
```
If you only want one of the folders to be deployed via `kustomize`, you will need to put them in separate deployment configurations.

> Note that besides setting `kustomize: true`, we also need to remove the `*` in our `manifests` array, because otherwise DevSpace CLI would try to use `kubectl apply -k` for all sub-folders of `manifests` and `more-manifests` instead of looking for the `kustomization.yaml` file within the `manifests` and the `more-manifests` folder.
