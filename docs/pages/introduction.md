---
title: Introduction to DevSpace
---

DevSpace allows developer teams to collaboratively build applications that seamlessly run and scale on Kubernetes.

## [What is DevSpace?](/docs/cli/what-is-devspace-cli)
DevSpace is an open-source command-line tool that enables your team to:
- **Build, test and debug applications directly inside Kubernetes**
- **Develop with hot reloading**: updates your running containers without rebuilding images or restarting containers
- **Unify deployment workflows** within your team and across dev, staging and production
- **Automate repetitive tasks** for image building and deployment

> [DevSpace](/docs/cli/what-is-devspace-cli) is a client-only, open-source dev tool for Kubernetes. It is <img width="20px" style="vertical-align: sub" src="/img/logos/github-logo.svg" alt="DevSpace Demo"> **[available on GitHub](https://github.com/devspace-cloud/devspace)** and works with any Kubernetes cluster because it simply uses your kube-context, just like kubectl or helm.

![DevSpace Workflow](/img/processes/workflow-devspace.png)

## [What is DevSpace Cloud?](/docs/cloud/what-is-devspace-cloud)
DevSpace Cloud is an optional add-on for DevSpace and allows developer teams to work together in shared dev clusters with:
- **Secure Multi-Tenancy & Namespace Isolation** ensuring that cluster users cannot break out of their namespaces
- **On-Demand Namespace Provisioning** allowing developers to create isolated namespaces with a single command
- **&gt;70% Cost Savings** using the sleep mode feature that automatically scales down pod replicas when users are not working

> [DevSpace Cloud](/docs/cloud/what-is-devspace-cloud) is the optional server-side component that DevSpace can connect to for creating isolated Kubernetes namespaces whenever a developer on your teams needs one. You can either
> - use the fully managed **[SaaS edition of DevSpace Cloud](https://app.devspace.cloud)**
> - or run it on your clusters using the <img width="20px" style="vertical-align: sub" src="/img/logos/github-logo.svg" alt="DevSpace Demo"> **[on-premise edition available on GitHub](https://github.com/devspace-cloud/devspace-cloud)**.

![DevSpace Cloud Workflow](/img/processes/workflow-devspace-cloud.png)
