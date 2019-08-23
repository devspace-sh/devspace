---
title: devspace add deployment
sidebar_label: deployment
---

```bash
#######################################################
############# devspace add deployment #################
#######################################################
Add a new deployment (docker image, components,
kubernetes manifests or helm chart) to your DevSpace configuration

Examples:
# Deploy a predefined component
devspace add deployment my-deployment --component=mysql
# Deploy a local dockerfile
devspace add deployment my-deployment --dockerfile=./Dockerfile
devspace add deployment my-deployment --image=myregistry.io/myuser/myrepo --dockerfile=frontend/Dockerfile --context=frontend/Dockerfile
# Deploy an existing docker image
devspace add deployment my-deployment --image=mysql
devspace add deployment my-deployment --image=myregistry.io/myusername/mysql
# Deploy local or remote helm charts
devspace add deployment my-deployment --chart=chart/
devspace add deployment my-deployment --chart=stable/mysql
# Deploy local kubernetes yamls
devspace add deployment my-deployment --manifests=kube/pod.yaml
devspace add deployment my-deployment --manifests=kube/* --namespace=devspace
#######################################################

Usage:
  devspace add deployment [deployment-name] [flags]

Flags:
      --chart string                                   A helm chart to deploy (e.g. ./chart or stable/mysql)
      --chart-repo string                              The helm chart repository url to use
      --chart-version string                           The helm chart version to use
      --component devspace list available-components   A predefined component to use (run devspace list available-components to see all available components)
      --context string
      --dockerfile string                              A dockerfile
  -h, --help                                           help for deployment
      --image string                                   A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)
      --manifests string                               The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)
      --namespace string                               The namespace to use for deploying
```
