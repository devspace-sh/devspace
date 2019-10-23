---
title: What are Component Deployments?
sidebar_label: Components
id: version-v4.1.0-what-are-components
original_id: what-are-components
---

Components are a convenience deployment method of DevSpace to deploy common Kubernetes resources without the hassle of defining complex kubernetes yamls or helm chart. A component is nothing else than a very dynamic Helm chart which allows you to specify Kubernetes resources as part of the Helm values for this chart.

You can define components in the `deployment` section of your `devspace.yaml`. A basic component looks like this:
```yaml
deployments:
- name: quickstart-nodejs
  helm:
    componentChart: true
    values:
      containers:
      - image: dscr.io/username/image
        resources:
          limits:
            cpu: "400m"
            memory: "500Mi"
```

Using `componentChart: true` is equivalent to the chart definition shown here:
```bash
helm:
  chart:                                # Helm chart to be deployed
    name: component-chart               # DevSpace component chart is a general-purpose Helm chart
    version: v0.0.6                     # This version is tied to the version of the DevSpace binary (allows to upgrade chart through the CLI)
    repo: https://charts.devspace.cloud
```
> The advantage of using `componentChart: true` is that DevSpace will automatically validate your `values` for this deployment and also upgrade these config options if they may change over time in a newer version of the chart.

## Types of components
There are two types of components:
- [Predefined components](../../../cli/deployment/components/configuration/overview-specification#devspace-add-deployment-name-component-mysql-redis)
- [Custom components](../../../cli/deployment/components/configuration/overview-specification#devspace-add-deployment-name-dockerfile-path)

### Predefined components
Predefined components allow you to add popular application components such as databases (e.g. mysql, mongodb) without having to manually define everything from scratch. DevSpace will ask you a couple of questions when [adding a predefined component](../../../cli/deployment/components/configuration/overview-specification#devspace-add-deployment-name-component-mysql-redis) and automatically configure everything for you. 

### Custom components
Custom components allow you to define components for your own application components or for additional applications which are not available as predefined component, yet.

## How are components deployed?
DevSpace deploys components using Helm. The chart that will be deployed is the [DevSpace Component Helm Chart](#devspace-component-helm-chart) and the values for the chart are specified directly under the `component` key in respective deployment within your `devspace.yaml`.

### DevSpace Component Helm Chart
The DevSpace Component Helm Chart is a general purpose Helm chart. Unlike other Helm charts which are designed to deploy one specific application (e.g. a mysql database), the DevSpace Component Helm Chart is more like a base chart that allows you to deploy any application. The benefits of using the DevSpace Component Helm Chart are:
- Much easier configuration than plain Kubernetes manifests
- Smart defaults and best practices (e.g. pods with persistent volumes will automatically be deployed as StatefulSets instead of Deployments)
- Much easier configuration than writing your own Helm chart
- Better deployment lifecycle management through Helm in comparison to plain Kubernetes manifests, including:
  - Tracking of Kubernetes resources of a deployment
  - Well-defined upgrade mechanism
  - Rollbacks when upgrades fail
  - Fast cleanup when removing deployments
