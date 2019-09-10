---
title: Deploying Kubernetes Manifests with kustomize
sidebar_label: kustomize
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

You can easily create a `kustomization.yaml` file within your `my-manifests` and within your `more-manifests` folder and tell DevSpace to deploy these manifest via `kustomize` by modifying the configuration as follows:
```yaml
deployments:
- name: my-deployment
  kubectl:
    manifests:
    - my-manifests/
    - more-manifests/
    kustomize: true
```
This configuration would tell DevSpace to deploy our application with the following commands:
```
kubectl apply -k manifests/
kubectl apply -k more-manifests/
```
If you only want one of the folders to be deployed via `kustomize`, you will need to put them in separate deployment configurations.

> Note that the `*` in the `manifests` section have been removed in the second config example which is using `kustomize: true`.
