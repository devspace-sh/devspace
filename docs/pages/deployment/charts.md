---
title: Configure Helm charts
---

By default, DevSpace CLI will copy the DevSpace Helm chart into the folder `chart/` within your project when you run `devspace init`. This chart is highly customizable and will make it much easier for you to get an enterprise-grade, scalable and secure deployment of your application running on Kubernetes.

See the following cusomization guides to:
- [Add packages to your Helm chart (e.g. database)](../charts/packages)
- [Configure persistent volumes](../charts/persistent-volumes)
- [Set environment variables](../charts/environment-variables)
- [Configure networking for your Helm chart (e.g. ingress)](../charts/networking)
- [Define multiple containers in your Helm chart](../charts/containers)
- [Add custom Kubernetes manifests (.yaml files)](../charts/custom-manifests)
- [Configure auto-scaling within your Helm Chart](../charts/scaling)
