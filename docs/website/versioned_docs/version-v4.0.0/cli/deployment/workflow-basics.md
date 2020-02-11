---
title: Workflow & Basics
id: version-v4.0.0-workflow-basics
original_id: workflow-basics
---

DevSpace fully automates the manual work of deploying Kubernetes manifests, Helm charts or components.

<br>
<img src="/img/processes/deployment-process-devspace.svg" alt="DevSpace Deployment Process" style="width: 100%;">

## Start Deployment
When you run one of the following commands, DevSpace will run the deployment process:
- `devspace deploy` (before deploying the application)
- `devspace dev` (before deploying the application and starting the development mode)

### Important Flags
The following flags are available for all commands that trigger the deployment process:
- `-b / --force-build` rebuild all images (even if they could be skipped because context and Dockerfile have not changed)
- `-d / --force-deploy` redeploy all deployments (even if they could be skipped because they have not changed)


## Deployment Process
DevSpace loads the `deployments` configuration from `devspace.yaml` and builds one deployment after another in the order that they are specified in the `deployments` array. Additionally, DevSpace also deploys related projects speficied in `dependencies`.


### 1. Build & Deploy Dependencies
DevSpace loads the `dependencies` section from the `devspace.yaml` and creates a dependency tree. The current project will represent the root of this tree. Based on this dependency tree, DevSpace will start from the leaves and run these steps for each dependency:
- Build images of the dependency as configured in the `images` section of the dependency's `devspace.yaml` (unless `skipBuild: true`)
- Deploy the dependency as configured in the `deployments` section of the dependency's `devspace.yaml`

[Learn more about deploying dependencies with DevSpace.](../../cli/deployment/advanced/dependencies)

> Dependencies allow you to deploy microservices, that the project you are currently deploying relies on. Dependencies can be located in a subpath of your project or they can be automatically loaded from a different git reporsitory.


### 2. Build, Tag & Push Images
DevSpace triggers the [image building process](../../cli/image-building/workflow-basics) for the images specified in the `images` section of the `devspace.yaml`.

[Learn more about image building with DevSpace.](../../cli/image-building/workflow-basics)


### 3. Tag Replacement
After finishing the image building process, DevSpace searches your deployments for references to the images that are specified in the `images` section of the `devspace.yaml`. If DevSpace finds that an image is used by one of your deployments and the deployment does not explicitly define a tag for the image, DevSpace will append the tag that has been auto-generated as part of the [automated image tagging](../../cli/image-building/workflow-basics#6-tag-image) during the image building process.

> To use automated tag replacement, make sure you do **not** specify image tags in the deployment configuration.

Replacing or appending tags to images that are used in your deployments makes sure that your deployments are always started using the most recently pushed image tag. This automated process saves a lot of time compared to manually replacing image tags each time before you deploy something.


### 4. Deploy Project
DevSpace iterates over every item in the `deployments` array defined in the `devspace.yaml` and deploy each of the deployments using the respective deployment tool:
- `kubectl` deployments will be deployed with `kubectl` (optionally using `kustomize` if `kustomize: true`)
- `helm` deployments will be deployed with the `helm` client that comes in-built with DevSpace
- `component` deployments will be deployed with the `helm` client that comes in-built with DevSpace

> Deployments with `kubectl` require `kubectl` to be installed.

> For `helm` and `component` deployments, DevSpace will automatically launch Tiller as a server-side component and setup RBAC for Tiller, so that it can only access the namespace it is deployed into.   
>   
> *We are waiting for Helm v3 to become stable, so we will not need to start a Tiller pod anymore to deploy Helm charts.*


## Useful Commands

### `devspace open`
To view your project in the browser either via port-forwarding or via ingress (domain), run the following command:
```bash
devspace open
```
When DevSpace asks you how to open your application, you have two options as shown here:
```bash
? How do you want to open your application?
  [Use arrows to move, space to select, type to filter]
> via localhost (provides private access only on your computer via port-forwarding)
  via domain (makes your application publicly available via ingress)
```
To use the second option, you either need to make sure the DNS of your domain points to your Kubernetes cluster and you have an ingress-controller running in your cluster OR you use [DevSpace Cloud](../../cloud/what-is-devspace-cloud), either in form of Hosted Spaces or by connecting your own cluster using the command `devspace connect cluster`.

> If your application does not open as exepected, run [`devspace analyze` and DevSpace will try to identify the issue](#devspace-analyze).

### `devspace analyze`
If your application is not starting as expected or there seems to be some kind of networking issue, you can let DevSpace run an automated analysis of your namespace using the following command:
```bash
devspace analyze
```
After analyzing your namespace, DevSpace compiles a report with potential issues, which is a good starting point for debugging and fixing issues with your deployments.

### `devspace list deployments`
To get a list of all deployments as well as their status and other information, run the following command:
```bash
devspace list deployments
```

### `devspace purge`
If you want to delete a deployment from kubernetes you can run:
```bash
# Removes all deployments remotely
devspace purge
# Removes deployment with given name
devspace purge --deployments=my-deployment-1,my-deployment-2
```

> Purging a deployment does not remove it from the `deployments` section in the `devspace.yaml`. It just removes the deployment from the Kubernetes cluster. To remove a deployment from `devspace.yaml`, run `devspace remove deployment [NAME]`.

### `devspace update dependencies`
If you are using dependencies from other git repositories, use the following command to update the cached git repositories of dependencies:
```bash
devspace update dependencies
```
