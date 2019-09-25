---
title: What is DevSpace?
sidebar_label: DevSpace
id: version-v4.0.0-what-is-devspace-cli
original_id: what-is-devspace-cli
---

DevSpace is an open-source command-line tool that enables your team to:
- **Build, test and debug applications directly inside Kubernetes**
- **Develop with hot reloading**: updates your running containers without rebuilding images or restarting containers
- **Unify deployment workflows** within your team and across dev, staging and production
- **Automate repetitive tasks** for image building and deployment

> DevSpace is a client-only, open-source dev tool for Kubernetes. It is <img width="20px" style="vertical-align: sub" src="/img/logos/github-logo.svg" alt="DevSpace Demo"> **[available on GitHub](https://github.com/devspace-cloud/devspace)** and works with any Kubernetes cluster because it simply uses your kube-context, just like kubectl or helm.


## Why DevSpace?
Building modern, distributed and highly scalable microservices with Kubernetes is hard - and it is even harder for large teams of developers. DevSpace is the next-generation tool for fast cloud-native software development.

<details>
<summary><h3>Standardize & Version Your Workflows</h3></summary>

DevSpace allows you to store all your workflows in one declarative config file: `devspace.yaml`
- **Codify workflow knowledge** about building images, deploying your project and its dependencies etc.
- **Version your workflows together with your code** (i.e. you can get any old version up and running with just a single command) 
- **Share your workflows** with your team mates

</details>

<details>
<summary><h3>Let Everyone on Your Team Deploy to Kubernetes</h3></summary>

DevSpace helps your team to standardize deployment and development workflows without requiring everyone on your team to become a Kubernetes expert.
- The DevOps and Kubernetes expert on your team can configure DevSpace using `devspace.yaml` and simply commits it via git
- If other developers on your team check out the project, they only need to run `devspace deploy` to deploy the project (including image building and deployment of dependencies, i.e. other related projects) to get a running instance of the project or they run `devspace dev` to start developing the project with an automated dev workflow
- The configuration of DevSpace is highly dynamic, so you can configure everything using [config variables](https://devspace.cloud/docs/cli/deployment/advanced/variables) that make it much easier to have one base configuration but still allow differences among developers (e.g. different sub-domains for testing)

> Giving everyone on your team on-demand access to a Kubernetes cluster is a challenging problem for system administrators and infrastructure managers. DevSpace Cloud, an optional add-on for DevSpace, makes sharing dev clusters much easier and safer. [Learn more about DevSpace Cloud](https://devspace.cloud/docs/cloud/what-is-devspace-cloud). 

</details>

<details>
<summary><h3>Speed Up Cloud-Native Development</h3></summary>

Instead of rebuilding images and redeploying containers, DevSpace allows you to **hot reload running containers while you are coding**:
- Simply edit your files with your IDE and see how your application reloads within the running container.
- The **high performance, bi-directional file synchronization** detects code changes immediately and synchronizes files immediately between your local dev environment and the containers running in Kubernetes
- Stream logs, connect debuggers or open a container terminal directly from your IDE with just a single command.

</details>

<details>
<summary><h3>Automate Repetitive Tasks</h3></summary>

Deploying and debugging services with Kubernetes requires a lot of knowledge and forces you to repeatedly run commands like `kubectl get po` and copy pod ids back and forth. Stop wasting time and let DevSpace automate the tedious parts of working with Kubernetes:
- DevSpace lets you build multiple images in parallel, tag them automatically and and deploy your entire application (including its dependencies) with just a single command
- Let DevSpace automatically start port-fowarding and log streaming, so you don't have to constantly copy and paste pod ids or run 10 commands to get everything started.

</details>

<details>
<summary><h3>Works with Any Kubernetes Clusters</h3></summary>

DevSpace is battle tested with many Kubernetes distributions including:
- **local Kubernetes clusters** like minikube, k3s, MikroK8s, kind
- **managed Kubernetes clusters** in GKE (Google Cloud), EKS (Amazon Web Service), AKS (Microsoft Azure), Digital Ocean
- **self-managed Kubernetes clusters** created with Rancher

> DevSpace also lets you switch seamlessly between clusters and namespaces. You can work with a local cluster as long as that is sufficient. If things get more advanced, you need cloud power like GPUs or you simply want to share a complex system such as Kafka with your team, simply tell DevSpace to use a remote cluster by switching your kube-context and continue working.

</details>


## Demo
<a href="https://youtu.be/G2l7VkQrkXo"><img width="100%" src="/img/devspace-cli-demo.gif" alt="DevSpace Demo"></a>

<p align="center">
<a href="https://youtu.be/G2l7VkQrkXo">Click here to watch the full-length video with explanations on YouTube [4min]</a>
</p>


## Features
Stop wasting time for running the same build and deploy commands over and over again. Let DevSpace automate your workflow with:
- [Automated Image Building](/docs/cli/image-building/workflow-basics) via `devspace build`
- [Automated Deployment](/docs/cli/deployment/workflow-basics) via `devspace deploy`
- [Efficient In-Cluster Development](/docs/cli/development/workflow-basics) via `devspace dev`

## How does it work?
DevSpace reads the configuration file `devspace.yaml` which you can simply generate for any of your project via `devspace init`. This config file allows you to define:
- [which Dockerfiles should be built](/docs/cli/image-building/configuration/overview-specification) (either with Docker, kaniko or with a custom build command), how to tag your images and where to store them (e.g. push to Docker Hub or to any private registry)
- [how your application should be deployed](/docs/cli/deployment/deployment-types) and with which tools (using [helm](/docs/cli/deployment/helm-charts/configuration/overview-specification), [kubectl](/docs/cli/deployment/kubernetes-manifests/configuration/overview-specification), [kustomize](/docs/cli/deployment/kubernetes-manifests/configuration/kustomize) or [components](/docs/cli/deployment/components/configuration/overview-specification))
- which [dependencies (related projects)](/docs/cli/deployment/advanced/dependencies) need to be deployed (e.g. a microservices from another git repository)
- [how your application should be developed within Kubernetes](/docs/cli/development/configuration/overview-specification) (e.g. configuring log streaming, terminal access, port fowarding, real-time file synchronization or remote debugging)

> **DevSpace is designed for teams** and its configuration is highly paramterizable, so that you can use dynamic [config variables](https://devspace.cloud/docs/cli/deployment/advanced/variables) within your `devspace.yaml`, commit the config via git together with the rest of your code and share your build, deployment and development workflows with your team mates.

![DevSpace Workflow](/img/processes/workflow-devspace.png)
