---
title: What are Kubernetes Manifest Deployments?
sidebar_label: Manifests (kubectl)
---

Kubernetes manifests are used to create, modify and delete Kubernetes resources such as pods, deployments, services or ingresses. It is very common to define manifests in form of `.yaml` files and send them to the Kubernetes API Server via commands such as `kubectl apply -f my-file.yaml` or `kubectl delete -f my-file.yaml`.

Learn more about how to [configure kubectl deployments](../../../cli/deployment/kubernetes-manifests/configuration/overview-specification). 

> In order to deploy kubernets manifests with DevSpace make sure you have `kubectl` installed and the manifests can be deployed via `kubectl apply -f my-file.yaml`.
