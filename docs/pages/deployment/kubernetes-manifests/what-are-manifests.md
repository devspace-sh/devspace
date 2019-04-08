---
title: What are Kubernetes manifests?
---

Kubernetes manifests are used to create, modify and delete Kubernetes resources such as pods, deployments, services or ingresses. It is very common to define manifests in form of `.yaml` files and send them to the Kubernetes API Server via commands such as `kubectl apply -f my-file.yaml` or `kubectl delete -f my-file.yaml`.

DevSpace CLI allows you to [add manifests to your set of deployments](/docs/deployment/kubernetes-manifests/add-manifests).
