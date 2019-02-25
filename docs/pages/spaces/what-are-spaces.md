---
title: What is a Space?
---

Spaces are smart Kubernetes namespaces which provide the following features:
- Automatic provisioning via `devspace create space [SPACE_NAME]`
- Automatic allocation of a subdomain for each Space, e.g. `my-app.devspace.host`
- Automatic RBAC configuration for better isolation of users
- Automatic resource limit configuration and enforcement
- Resource auto-scaling within the configured limits
- Smart analysis of issues within your Space via `devspace analyze`

You can create Spaces that either run on DevSpace.cloud (will get a `.devspace.host` subdomain) or on your own Kubernetes clusters (external Spaces with an automatically provisioned subdomain of one of your domains).
