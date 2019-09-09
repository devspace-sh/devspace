---
title: Workflow & Basics
---

DevSpace fully automates the manual work of deploying Kubernetes manifests, Helm charts or components.

<br>
<img src="/img/processes/deployment-process-devspace.svg" alt="DevSpace Deployment Process" style="width: 100%;">

## Commands Triggering Image Building
When you run one of the following commands, DevSpace will run the deployment process:
- `devspace deploy` (before deploying the application)
- `devspace dev` (before deploying the application and starting the development mode)

### Important Flags
The following flags are available for all commands that trigger image building:
- `-b / --force-build` rebuild all images (even if they could be skipped because context and Dockerfile have not changed)
- `-d / --force-deploy` redeploy all deployments (even if they could be skipped because they have not changed)

## Deployment Process
DevSpace loads the `images` configuration from `devspace.yaml` and builds all images in parallel. The multi-threded, parallel build process of DevSpace speeds up image building drastically, especially when building many images and using remote build methods. 

> You can use the `--build-sequential` flag to tell DevSpace to build images sequentially instead of using the parallel approach.


### 1. Build & Deploy Dependencies
DevSpace loads the `dependencies` section from the `devspace.yaml` and creates a dependency tree. The current project will be the root of the tree. Based on this dependency tree, DevSpace will start from the leaves and run these steps for each dependency:
- Build images of the dependency as configured in the `images` section of the dependency's `devspace.yaml` (unless `skipBuild: true`)
- Deploy the dependency as configured in the `deployments` section of the dependency's `devspace.yaml`

[Learn more about deploying dependencies with DevSpace](/docs/cli/deployment/advanced/dependencies)

> Dependencies allow you to deploy microservices from other git repositories, that the project you are currently deploying relies on.


### 2. Build, Tag & Push Images
DevSpace triggers the [image building process](/docs/cli/image-building/workflow-basics) for the images specified in the `images` section of the `devspace.yaml`.


### 3. Tag Replacement
After finishing the image building process, DevSpace searches your deployments for references to the images that are specified in the `images` section of the `devspace.yaml`. If DevSpace finds that an image is used by one of your deployments and the deployment does not explicitly define a tag for the image, DevSpace will append the tag that has been auto-generated accroding to the [tagging strategy](#TODO) as part of the image building process.

> To use automated tag replacement, make sure you do not speficy an image tag when using an image in your deployment configuration.

Replacing or appending tags to images that are used in your deployments makes sure that your deployments are always started using the most recenltly pushed image tag. This automated process saves a lot of time compared to manually replacing image tags each time before you deploy something.


### 4. Deploy Project
DevSpace will iterate over every item in the `deployments` array defined in the `devspace.yaml` and deploy each deployment using the respective deployment tool:
- `kubectl` deployments will be deployed with `kubectl` (optionally using `kustomize` if `kustomize: true`)
- `helm` deployments will be deployed with the `helm` client that comes in-built with DevSpace
- `component` deployments will be deployed with the `helm` client that comes in-built with DevSpace

> Deployments with `kubectl` require `kubectl` to be installed.

> For `helm` and `component` deployments, DevSpace will automatically launch Tiller as a server-side component and setup RBAC for Tiller, so that it can only access the namespace it is deployed into.   
> *We are waiting for Helm v3 to become stable, so we will not need to start a Tiller pod anymore to deploy Helm charts.*
