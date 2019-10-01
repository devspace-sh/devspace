---
title: What is a Space?
id: version-v3.5.18-what-are-spaces
original_id: what-are-spaces
---

Spaces allow teams to develop together in a single kubernetes cluster. In essence spaces are **isolated kubernetes namespaces** and developers can create them whenever they need them. 

> DevSpace CLI automatically configures their kube context locally so they are able to access kubernetes directly with DevSpace CLI and all their other favourite tools like kubectl, helm and kustomize.  

DevSpace Cloud automatically sets up RBAC, resource quotas, network policies, pod security policies etc. to isolate these namespaces and makes sure that developers stay within the borders of their Spaces. Administrators are able to [configure everything](../../cloud/spaces/resource-limits) from limits for computing resources to custom resource templates that will be deployed automatically during space creation.  

## Why use a space?

Spaces have the following benefits:
- Automatic provisioning via `devspace create space [SPACE_NAME]`
- Automatic local kube context configuration
- Automatic allocation of a subdomain for each Space, e.g. `my-app.devspace.host` and ingress creation via `devspace open`
- Automatic RBAC configuration for better isolation of users
- Automatic resource limit configuration and enforcement
- Resource auto-scaling within the configured limits
- Automatic deployment of predefined manifests on space creation
- Vast majority of options how to configure a Space default [settings](../../cloud/spaces/resource-limits)

You can create Spaces that either run on DevSpace Cloud (will get a `.devspace.host` subdomain) or on [your own Kubernetes clusters](../../cloud/clusters/connect) (external Spaces with an automatically provisioned subdomain of one of your domains).
