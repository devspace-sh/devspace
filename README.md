<img src="docs/static/media/logos/devspace-logo-primary.svg" width="600">

### **[Website](https://devspace.sh)** • **[Quickstart](#quickstart)** • **[Documentation](https://devspace.sh/cli/docs/introduction)** • **[Blog](https://loft.sh/blog)** • **[Twitter](https://twitter.com/devspace)**

![Build Status Passing](https://img.shields.io/github/workflow/status/loft-sh/devspace/Test%20&%20Release%20CLI%20Version/main?style=for-the-badge)
![Latest Release](https://img.shields.io/github/v/release/loft-sh/devspace?style=for-the-badge&label=Latest%20Release&color=%23007ec6)
![License: Apache-2.0](https://img.shields.io/github/license/loft-sh/devspace?style=for-the-badge&color=%23007ec6)
![Total Downloads (GitHub Releases)](https://img.shields.io/github/downloads/loft-sh/devspace/total?style=for-the-badge&label=Total%20Downloads&color=%23007ec6)
![NPM Installs per Month](https://img.shields.io/npm/dm/devspace?label=NPM%20Installs&style=for-the-badge&color=%23007ec6)
![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/6945/badge)

[![Join us on Slack!](docs/static/img/slack.svg)](https://slack.loft.sh/)

### Client-Only Developer Tool for Cloud-Native Development with Kubernetes
- **Build, test and debug applications directly inside Kubernetes**
- **Develop with hot reloading**: updates your running containers without rebuilding images or restarting containers
- **Unify deployment workflows** within your team and across dev, staging and production
- **Automate repetitive tasks** for image building and deployment

<br>

![DevSpace Compatibility](docs/static/img/cluster-compatibility.png)

<br>

<p align="center">
⭐️ <strong>Do you like DevSpace? Support the project with a star</strong> ⭐️
</p>

<br>

DevSpace was created by [Loft Labs](https://loft.sh) and is a [Cloud Native Computing Foundation (CNCF) sandbox project](https://www.cncf.io/sandbox-projects/).

<br>

## Contents
- [Why DevSpace?](#why-devspace)
- [Quickstart Guide](#quickstart)
- [Architecture & Workflow](#architecture--workflow)
- [Contributing](#contributing)
- [FAQ](#faq)

<br>

## Why DevSpace?
Building modern, distributed and highly scalable microservices with Kubernetes is hard - and it is even harder for large teams of developers. DevSpace is the next-generation tool for fast cloud-native software development.

<details>
<summary><b>Standardize & Version Your Workflows</b></summary>
<br>

DevSpace allows you to store all your workflows in one declarative config file: `devspace.yaml`
- **Codify workflow knowledge** about building images, deploying your project and its dependencies etc.
- **Version your workflows together with your code** (i.e. you can get any old version up and running with just a single command)
- **Share your workflows** with your team mates

<br>
</details>

<details>
<summary><b>Let Everyone on Your Team Deploy to Kubernetes</b></summary>
<br>

DevSpace helps your team to standardize deployment and development workflows without requiring everyone on your team to become a Kubernetes expert.
- The DevOps and Kubernetes expert on your team can configure DevSpace using `devspace.yaml` and simply commits it via git
- If other developers on your team check out the project, they only need to run `devspace deploy` to deploy the project (including image building and deployment of other related project etc.) and they have a running instance of the project
- The configuration of DevSpace is highly dynamic, so you can configure everything using [config variables](https://devspace.sh/cli/docs/configuration/variables/basics) that make it much easier to have one base configuration but still allow differences among developers (e.g. different sub-domains for testing)

> Giving everyone on your team on-demand access to a Kubernetes cluster is a challenging problem for system administrators and infrastructure managers. If you want to efficiently share dev clusters for your engineering team, take a look at [www.loft.sh](https://loft.sh/).

<br>
</details>

<details>
<summary><b>Speed Up Cloud-Native Development</b></summary>
<br>

Instead of rebuilding images and redeploying containers, DevSpace allows you to **hot reload running containers while you are coding**:
- Simply edit your files with your IDE and see how your application reloads within the running container.
- The **high performance, bi-directional file synchronization** detects code changes immediately and synchronizes files immediately between your local dev environment and the containers running in Kubernetes
- Stream logs, connect debuggers or open a container terminal directly from your IDE with just a single command.

<br>
</details>

<details>
<summary><b>Automate Repetitive Tasks</b></summary>
<br>

Deploying and debugging services with Kubernetes requires a lot of knowledge and forces you to repeatedly run commands like `kubectl get pod` and copy pod ids back and forth. Stop wasting time and let DevSpace automate the tedious parts of working with Kubernetes:
- DevSpace lets you build multiple images in parallel, tag them automatically and and deploy your entire application (including its dependencies) with just a single command
- Let DevSpace automatically start port-fowarding and log streaming, so you don't have to constantly copy and paste pod ids or run 10 commands to get everything started.

<br>
</details>

<details>
<summary><b>Works with Any Kubernetes Clusters</b></summary>
<br>

DevSpace is battle tested with many Kubernetes distributions including:
- **local Kubernetes clusters** like minikube, k3s, MikroK8s, kind
- **managed Kubernetes clusters** in GKE (Google Cloud), EKS (Amazon Web Service), AKS (Microsoft Azure), Digital Ocean
- **self-managed Kubernetes clusters** created with Rancher

> DevSpace also lets you switch seamlessly between clusters and namespaces. You can work with a local cluster as long as that is sufficient. If things get more advanced, you need cloud power like GPUs or you simply want to share a complex system such as Kafka with your team, simply tell DevSpace to use a remote cluster by switching your kube-context and continue working.

<br>
</details>

<br>

## Quickstart

Please take a look at our [getting started guide](https://devspace.sh/docs/getting-started/installation).

<br>

## Architecture & Workflow
![DevSpace Workflow](docs/static/img/workflow-devspace.png)

DevSpace runs as a single binary CLI tool directly on your computer and ideally, you use it straight from the terminal within your IDE. DevSpace does not require a server-side component as it communicates directly to your Kubernetes cluster using your kube-context, just like kubectl.

<br>

## Contributing

Help us make DevSpace the best tool for developing, deploying and debugging Kubernetes apps.

[![Join us on Slack!](docs/static/img/slack.svg)](https://slack.loft.sh/)

### Reporting Issues

If you find a bug while working with the DevSpace, please [open an issue on GitHub](https://github.com/loft-sh/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

### Feedback & Feature Requests

You are more than welcome to open issues in this project to:

- [Give feedback](https://github.com/loft-sh/devspace/issues/new?labels=kind%2Ffeedback&title=Feedback:)
- [Suggest new features](https://github.com/loft-sh/devspace/issues/new?labels=kind%2Ffeature&template=feature-request.md&title=Feature%20Request:)
- [Report Bugs](https://github.com/loft-sh/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug%20Report:)

### Contributing Code

This project is mainly written in Golang. If you want to contribute code:

1. Ensure you are running golang version 1.11.4 or greater for go module support
2. Set the following environment variables:
   ```
   GO111MODULE=on
   GOFLAGS=-mod=vendor
   ```
3. Check-out the project: `git clone https://github.com/loft-sh/devspace && cd devspace`
4. Make changes to the code
5. Build the project, e.g. via `go build -o devspace[.exe]`
6. Evaluate and test your changes `./devspace [SOME_COMMAND]`

See [Contributing Guidelines](CONTRIBUTING.md) for more information.

The DevSpace project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).

<br>

## FAQ

<details>
<summary>What is DevSpace?</summary>

DevSpace is an open-source command-line tool that provides everything you need to develop, deploy and debug applications with Docker and Kubernetes. It lets you streamline deployment workflows and share them with your colleagues through a declarative configuration file `devspace.yaml`.

</details>

<details>
<summary>Is DevSpace free?</summary>

**YES.** DevSpace is open-source and you can use it for free for any private projects and even for commercial projects.

</details>

<details>
<summary>Do I need a Kubernetes cluster to use DevSpace?</summary>

**Yes.** You can either use a local cluster such as Docker Desktop Kubernetes, minikube, or Kind, but you can also use a remote cluster such as GKE, EKS, AKS, RKE (Rancher), or DOKS.

</details>

<details>
<summary>Can I use DevSpace with my existing Kubernetes clusters?</summary>

**Yes.** DevSpace is using your regular kube-context. As long as you can run `kubectl` commands with a cluster, you can use this cluster with DevSpace as well.

</details>

<details>
<summary>What is a Helm chart?</summary>

[Helm](https://helm.sh/) is the package manager for Kubernetes. Packages in Helm are called Helm charts.

</details>

<br>
<br>

## License

DevSpace is released under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.

DevSpace is a [Cloud Native Computing Foundation (CNCF) sandbox project](https://www.cncf.io/sandbox-projects/) and was contributed by [Loft Labs](https://www.loft.sh).

<div align="center">
    <img src="https://raw.githubusercontent.com/cncf/artwork/master/other/cncf-sandbox/horizontal/color/cncf-sandbox-horizontal-color.svg" width="300" alt="CNCF Sandbox Project">
</div>
