---
title: Build & deploy Dockerfiles
id: version-v3.5.18-dockerfiles
original_id: dockerfiles
---

DevSpace CLI lets you easily build Dockerfiles and define deployments for the Docker images created from such Dockerfiles. This will use the easy to use deployment method [component](../../deployment/components/what-are-components). There are also other options how you can deploy your Dockerfile e.g. with [kubernetes manifests](../../deployment/kubernetes-manifests/what-are-manifests) or a [helm chart](../../deployment/helm-charts/what-are-helm-charts)

### Add deployments for existing Dockerfiles
Run one of the following commands to add a custom component to your deployments defined in `devspace.yaml` based on an existing Dockerfile:
```bash
devspace add deployment [deployment-name] --dockerfile=""
devspace add deployment [deployment-name] --dockerfile="" --image="my-registry.tld/[username]/[image]"
```

The difference between the first command and the second one is that the second one specifically defines where the Docker image should be pushed to after building the Dockerfile. In the first command, DevSpace CLI would assume that you want to use the [DevSpace Container Registry](../../cloud/images/dscr-io) provided by DevSpace Cloud.

> If you are using a private Docker registry, make sure to [login to this registry](../../image-building/registries/authentication).

After adding a new deployment, you need to manually redeploy in order to start the newly added component together with the remainder of your previouly existing deployments.
```bash
devspace deploy
```
