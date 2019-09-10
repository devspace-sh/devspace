---
title: What are Component Deployments?
sidebar_label: Components
---

Components are a convenience deployment method of DevSpace to deploy common kubernetes resources without the hassle of defining complex kubernetes yamls. You can define components in the `deployment` section of your `devspace.yaml`. A basic component looks like this:
```yaml
deployments:
- name: quickstart-nodejs
  component:
    containers:
    - image: dscr.io/username/image
      resources:
        limits:
          cpu: "400m"
          memory: "500Mi"
```

## Types of components
There are two types of components:
- [Predefined components](/docs/deployment/components/add-predefined-components)
- [Custom components](/docs/deployment/components/add-custom-components)

### Predefined components
Predefined components allow you to add popular application components such as databases (e.g. mysql, mongodb) without having to manually define everything from scratch. DevSpace will ask you a couple of questions when [adding a predefined component](/docs/deployment/components/add-predefined-components) and automatically configure everything for you. 

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
