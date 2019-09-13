---
title: What are Kubernetes manifests?
id: version-v3.5.18-what-are-manifests
original_id: what-are-manifests
---

Kubernetes manifests are used to create, modify and delete Kubernetes resources such as pods, deployments, services or ingresses. It is very common to define manifests in form of `.yaml` files and send them to the Kubernetes API Server via commands such as `kubectl apply -f my-file.yaml` or `kubectl delete -f my-file.yaml`.

DevSpace CLI is able to deploy kubernetes manifests (see [add manifests to your set of deployments](/docs/deployment/kubernetes-manifests/add-manifests)). In order to deploy kubernets manifests with DevSpace make sure you have `kubectl` installed and the manifests can be deployed via `kubectl apply -f my-file.yaml`.
