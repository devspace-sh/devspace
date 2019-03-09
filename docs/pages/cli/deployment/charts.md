---
title: Configure Helm charts
---

By default, DevSpace CLI will copy the DevSpace Helm Chart into the folder `chart/` within your project when you run `devspace init`. This chart is highly customizable and will make it much easier for you to get an enterprise-grade, scalable and secure deployment of your application running on Kubernetes.

See the following cusomization guides to:
- [Add packages to your Helm chart (e.g. database)](/docs/chart/customization/packages)
- [Configure persistent volumes](/docs/chart/customization/persistent-volumes)
- [Set environment variables](/docs/chart/customization/environment-variables)
- [Configure networking for your Helm chart (e.g. ingress)](/docs/chart/customization/networking)
- [Define multiple containers in your Helm chart](/docs/chart/customization/containers)
- [Add custom Kubernetes manifests (.yaml files)](/docs/chart/customization/custom-manifests)
- [Configure auto-scaling within your Helm Chart](/docs/chart/customization/scaling)
