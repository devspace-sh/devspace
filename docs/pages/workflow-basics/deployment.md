---
title: Deploy with DevSpace
---

DevSpace CLI deploys your application by iterating over the `deployments` array defined in your DevSpace configuration.

Running `devspace deploy` will do the following:
1. Build all Docker [images that you specified in `images` within `.devspace/config.yaml`](/docs/cli/deployment/images)
2. Push the Docker images to any [Docker registry](/docs/cli/images/workflow)
3. Deploy all [deployments defined in `.devspace/config.yaml`](/docs/cli/deployment/deployments)

## Default deployment created by `devspace init`
When running `devspace init` within your project, DevSpace CLI defines a deployment called `default` within your config file `.devspace/config.yaml`.
```yaml
deployments:
- name: default
  helm:
    chartPath: ./chart
```
This `default` deployment is configured to deploy the Helm chart locaed in `./chart`. When running `devspace init`, the [DevSpace Helm Chart](/docs/chart/basics/devspace-helm-chart) will will automatically be added into the `./chart` folder of your project.

Unlike `images`, `deployments` is an array and not a map because DevSpace CLI will iterate over the deployment one after another. It has been designed this way because the order in which your deployments are starting might be relevant depending on your application.

## Add additonal deployments
If you want to deploy another Helm chart in your project, simply use the `devspace add deployment [NAME]` command.
```bash
devspace add deployment database --chart=./db/chart
```

The command shown above would add a new deployment to your DevSpace configuration. The resulting configuration would look similar to this one:

```yaml
deployments:
- name: default
  helm:
    chartPath: ./chart
- name: database
  helm:
    chartPath: ./db/chart
```

## Remove a deployment
Instead of manually removing a deployment from your configuration file, you can also use the `devspace remove deployment` command.
```bash
devspace remove deployment database
```
The command shown above would remove the deployment with name `database` from your DevSpace configuration.

## Deploy with kubectl instead of Helm
Instead of using your local Docker daemon to build your images, you can also use [kaniko](https://github.com/GoogleContainerTools/kaniko) to build Docker images. Using kaniko has the advantage that you are building the image inside a container that runs remotely on top of Kubernetes. Using DevSpace Cloud, this container would run inside the Space that you are currently working with.
```yaml
deployments:
- name: default
  helm:
    chartPath: ./chart
- name: database
  kubectl:
    manifests:
    - ./db/manifests/*
    - ./db/rbac.yaml
```
The config excerpt shown above would tell DevSpace CLI to deploy every Kubernetes manifest in `./db/manifests` as well as the manifest contained in the file `./db/rbac.yaml`.

> **It is recommended to use Helm instead of kubectl for deployment.** To add plain manifests, you can 
[use the DevSpace Helm Chart](/docs/chart/basics/devspace-helm-chart) and then
[add custom Kubernetes manifests](/docs/chart/customization/custom-manifests).
