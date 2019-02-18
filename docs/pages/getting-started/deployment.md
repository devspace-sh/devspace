---
title: 3. Deploy with DevSpace
---

To be able to deploy applications with DevSpace.cli, you first need to create a so-called Space.
> Spaces are smart Kubernetes namespaces. You can create Spaces that either run on DevSpace.cloud or on your own Kubernetes clusters. [Learn more about Spaces.](../spaces/what-are-spaces)

## Create a Space
With the following command, you can create a Space called `production` for your project:
```bash
devspace create space production
```
You can create multiple Spaces for your project (e.g. production, staging, development). DevSpace.cli will automatically work with the Space that you created last. To actively switch to a Space, you can use the command: `devspace use space [name]`

Learn more about [working with multiple Spaces](../spaces/switch-spaces).

## Deploy your application
Now, you can deploy your application to your `production` Space with the following command:
```bash
devspace deploy production
```
This command will do the following:
1. Build a [Docker image](../deployment/images) as defined in the `Dockerfile`
2. Push this Docker image (either to [DevSpace Container Registry (dscr.io)](../images/internal-registry) or to any [external registry](../images/external-registry))
3. Deploy your [Helm chart](../charts/what-are-helm-charts) as defined in `chart/`
4. Make your application available on a `.devspace.host` domain

## Access your application
After deploying your application, you can access it on `my-space-url.devspace.host`. Take a look at the last couple of lines of the `devspace deploy` output to find the auto-generated URL of your Space:
```bash
DEVSPACE DEPLOY OUTPUT WITH URL
```

<details>
<summary>
## Connect a custom domain
</summary>


</details>

<details>
<summary>
## Access your Space with kubectl
</summary>
Spaces can be used very much like any regular Kubernetes namespace. Therefore, you can run any `kubectl` command within your Space. This lets you manually access, debug or modify Kubernetes resources.

<details>
<summary>
### Install kubectl
</summary>


</details>

### Useful kubectl commands
Here is a list of common kubectl commands:

#### View all pods (group of containers) in your Space
```bash
kubectl get pods
```

> Pods are groups of containers that share a network stack. [Learn more about pods](../kubernetes/pods)

#### View all services in your Space
```bash
kubectl get services
```

</details>


<details>
<summary>
## Troubleshooting
</summary>

If you get an HTTP error when accessing your Space, the following guides can help you solve the most common issues:

### 404 Not Found

### 502 Bad Gateway

### 503 Service Unavailable

### 504 Gateway Timeout

### 500 Internal Server Error

</details>


## Learn more about deploying with DevSpace


See the following guides to learn more:
- [Connect custom domains](../deployment/domains)
- [Monitor and debug deployed applications](../deployment/debugging)
- [Scale deployed applications](../deployment/scaling)
- [Configure Docker image](../deployment/images)
- [Configure Helm chart](../deployment/charts)
