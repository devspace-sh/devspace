---
title: Configuring Automatic Reload of Deployments
sidebar_label: Auto-Reloading
---

There are certain use cases where you want to rebuild and redeploy the whole application instead of using the file synchronization and hot reloading. DevSpace provides you the options to specify special paths that are watched during `devspace dev` and any change to such a path will trigger a redeploy.  

Auto-reloading can be configured in the `dev.autoReload` section of `devspace.yaml`.
```yaml
images:
  backend:
    image: john/devbackend
  database:
    image: john/database
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/devbackend
- name: custom-manifests
  kubectl:
    manifests:
    - manifests/*
    - more/manifests/*
dev:
  autoReload:
    paths:
    - ./package.json
    - ./important-config-files/*
    images:
    - database
    deployments:
    - custom-manifests
```

Take a look at the [redeploy-instead-of-hot-reload exmaple](https://github.com/devspace-cloud/devspace/tree/master/examples/redeploy-instead-of-hot-reload) to see how to disable hot reloading at all and enable redeployment on every file change instead.

## `dev.autoReload.paths`
The `dev.autoReload.paths` option expects an array of strings with paths that should be watched for changes. If a changes occurs in any of the specified paths, DevSpace will stop the development mode, rebuild the images, redeploy the application and restart the devepment mode afterwards.

## `dev.autoReload.images`
The `dev.autoReload.images` option expects an array of strings with image names from the `images` section of the `devspace.yaml`. If a changes occurs to the `dockerfile` or to one of the files within the `context` of this image, DevSpace will stop the development mode, rebuild the images, redeploy the application and restart the devepment mode afterwards.

## `dev.autoReload.deployments`
The `dev.autoReload.deployments` option expects an array of strings with names of deployments from the `deployments` section of the `devspace.yaml`. If a changes occurs to any of the files that belong to this deployment, DevSpace will stop the development mode, rebuild the images, redeploy the application and restart the devepment mode afterwards.

> For `kubectl` deployments, DevSpace watches for all paths configured under `manifests`.

> For `helm` deployments, DevSpace watches for changes in the `valuesFiles` or changes in the chart path of a local chart (configured as `chart.name`).

> For `component` deployments, this option is not doing anything.
