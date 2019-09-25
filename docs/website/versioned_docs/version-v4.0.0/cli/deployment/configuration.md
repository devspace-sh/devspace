---
title: Configure Deployments
sidebar_label: Configuration
id: version-v4.0.0-configuration
original_id: configuration
---

Deployments are configured within the `deployments` section of the `devspace.yaml`.
```yaml
# An array of deployments (kubectl, helm, component) which will be deployed with DevSpace in the specified order
deployments:
- name: deployment-1                    # Name of this deployment
  component: ...                        # Deploy a Component
- name: deployment-2                    # Name of this deployment
  kubectl: ...                          # Deploy Kubernetes manifests or Kustomizations (using kubectl and kustomize)
- name: deployment-3                    # Name of this deployment
  helm: ...                             # Deploy a Helm Chart
- name: deployment-4                    # Name of this deployment
  helm: ...                             # Deploy another Helm Chart
...
```

> Unlike images which are build in parallel, deployments will be deployed sequentially following the order in which they are specified in the `devspace.yaml`.

## Config Options
The following config options exist for every deployment:
- `name` stating the name of the deployment (required)
- `component` for [**Configuring Component Deployments**](../../cli/deployment/components/configuration/overview-specification)
- `kubectl` for [**Configuring Manifest Deployments**](../../cli/deployment/kubernetes-manifests/configuration/overview-specification)
- `helm` for [**Configuring Helm Chart Deployments**](../../cli/deployment/helm-charts/configuration/overview-specification)
- `namespace` stating a namespace to deploy to (optional, see note below)

> **Note:** Use `namespace` **only** if you want to run a deployment in another namespace than the remaining deployments. Generally, DevSpace uses the default namespace of the current kube-context and runs all deployments in the same namespace.

> You **cannot** use `component`, `helm` and `kubectl` in combination. You must specify **exactly one** of the three. 
