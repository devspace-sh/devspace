---
title: /chart (Helm Chart)
---

Helm is the most popular package manager for Kubernetes. With Helm, it is easy to install and maintain packages (so-called helm charts) within a Kubernetes cluster. The DevSpace CLI uses Helm to start DevSpaces and container registries. Helm currently requires a server-side component called Tiller. The DevSpace CLI installs a Tiller server if you don't have one already.

When running `devspace init` or `devspace up` for the first time, the DevSpace CLI will create an initial helm chart inside your project. This helm chart is located in the [chart/](#) folder within your project. You can customize the chart to your needs. The default helm chart contains the following files:

```bash
chart/
|-- Chart.yaml              # Chart definition
|-- values.yaml             # Deployment variables
|-- templates/              # Template folder
|   |-- deployment.yaml     # deployment definition
|   |-- service.yaml        # service definition
|   |-- ingress.yaml        # ingress definition
|   |-- ...                 # further templates can be added
```

During the deployment process, the values defined in the [values.yaml](#) are used to create Kubernetes resources based on the templates defined in [templates/](#). After filling the templates with the variables, the chart is deployed to the Kubernetes cluster.

**Note: You don't need to install Helm or Tiller, to use the DevSpace CLI.**
