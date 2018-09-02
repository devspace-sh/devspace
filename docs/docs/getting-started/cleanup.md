---
title: 4. Cleanup
---

## devspace down
Use the command `devspace down` to shutdown your DevSpace (i.e. remove the release deployed with helm from your Kubernetes cluster).

## devspace reset
Use `devspace reset` to reset your project to its original state. This will:
1. shutdown your DevSpace (i.e. `devspace down`),
2. remove the Docker registry from your Kubernetes cluster,
3. remove the Tiller server from your Kubernetes cluster,
4. and remove the [.devspace/](/docs/configuration/config.yaml.html) folder, the [chart/](/docs/configuration/chart.html) folder and the [/Dockerfile](/docs/configuration/dockerfile.html) from your project.
