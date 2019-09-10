---
title: Configure Deployments
sidebar_label: Configuration
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

How to configure a single deployment depends on the type of the deployment:
- [**Component Configuration**](/docs/cli/deployment/components/configuration/overview-specification)
- [**Manifest Configuration**](/docs/cli/deployment/kubernetes-manifests/configuration/overview-specification)
- [**Helm Chart Configuration**](/docs/cli/deployment/helm-charts/configuration/overview-specification)
