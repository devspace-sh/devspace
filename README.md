[![DevSpace Logo](docs/website/static/img/github-readme-header.svg)](https://devspace.cloud/)
---

[Website](https://devspace.cloud/) • 
[Documentation](https://devspace.cloud/docs) • 
[Blog](https://devspace.cloud/blog) • 
[Slack](https://devspace.cloud/slack)

[![Build Status](https://travis-ci.org/devspace-cloud/devspace.svg?branch=master)](https://travis-ci.org/devspace-cloud/devspace)
[![Go Report Card](https://goreportcard.com/badge/github.com/devspace-cloud/devspace)](https://goreportcard.com/report/github.com/devspace-cloud/devspace)
[![Slack](https://devspace.cloud/slack/badge.svg)](http://devspace.cloud/slack)
[![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/home?status=Just%20found%20out%20about%20%23DevSpace.cli%3A%20https%3A//github.com/devspace-cloud/devspace%0A%0AIt%20lets%20you%20build%20cloud%20native%20software%20directly%20on%20top%20of%20%23Kubernetes%20and%20%23Docker%0A%23CloudNative%20%23k8s)


**DevSpace accelerates developing, deploying and debugging applications with Docker and Kubernetes.**

## How does it work?

<Details>
<Summary><b>Containerize</b> any project in minutes</Summary>

### Initialize your project
```
devspace init
```
#### DevSpace uses smart defaults for many programming languages and frameworks to:
1. Automatically create a Dockerfile for your app
2. Add a highly customizable Helm chart to your project

> If you already have a Dockerfile or a Helm chart, DevSpace.cli will ask you if you want to use them instead of the default files.

Customize Dockerfile and Kubernetes deployment:
 - [Add packages (e.g. databases)](#)
 - [Configure persistent volumes](#)
 - [Set environment variables](#)
 - [Enable auto-scaling](#)

---

</Details>

<Details>
<Summary><b>Deploy</b> containerized applications with ease</Summary>

### 1. Create a Space
```
devspace create space my-app
```
If you are using DevSpace.cloud, you can create Spaces with a single command. Spaces are smart Kubernetes namespaces with:
- Automatic allocation of a subdomain for each Space
- Automatic RBAC configuration for better isolation of users
- Resource auto-scaling within the configured limits
- [and much more...](#)

> **If you do not want to use DevSpace.cloud, you will not be able to create Spaces.** You can skip this step and deploy your application to a regular Kubernetes namespace using `devspace deploy`.

### 2. Deploy your application
```
devspace deploy
```

#### What does `devspace deploy` do?
1. Builds, tags and pushes your Docker images
2. Creates pull secrets for your image registries
3. Deploys your project with the newest images

### 3. Access your application
After deploying, your application will run on a domain that is connected to your Space:
- **auto-generated, e.g. `my-app.devspace.host`**
- **custom domain ([Learn how to connected a custom domain](#))**

> **If you are not using DevSpace.cloud, you will need to manually configure a domain and an ingress.**

---

</Details>

<Details>
<Summary><b>Develop</b> cloud-native software faster then ever</Summary>

### Develop in a production-like environment
```
devspace dev
```
**With DevSpace, you can build and test your application directly inside Kubernetes.** Thanks to our real-time code sync, you can even use hot reloading tools (e.g. nodemon) to refresh your running application without having to waste time on re-building and re-deploying your application every time you change your code. With DevSpace, your containers are updated in real-time without any delay.

Learn more about development with DevSpace:
- [Real-time code synchronization for hot reloading](#)
- [Automatic port forwarding for access via localhost](#)
- [Terminal proxy for running commands in your containers](#)

---

</Details>

<Details>
<Summary><b>Debug</b> deployments without hassle</Summary>

### Speed up finding and solving issues
```
devspace analyze
```
**DevSpace automatically analyzes your deployments**, identifies potential issues and helps you resolve them:
- Identify reasons for image pull failure
- View log snapshots of crashed containers
- Debug networking issues (e.g. misconfigured services)

Learn more about development with DevSpace:
- [Automate issue detection with DevSpace](#)
- [Stream container logs with DevSpace](#)
- [Use the debugger of your IDE with DevSpace](#)
- [Start terminal sessions for debugging](#)

</Details>

<br>

![DevSpace.cli Demo](https://github.com/devspace-cloud/devspace/raw/master/docs/website/static/img/devspace-cli-demo-readme.gif)


## Getting started with DevSpace
### 1. Install DevSpace.cli

<Details>
<Summary><b>via Windows Powershell</b></Summary>

```
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]'Tls,Tls11,Tls12'
md -Force "$Env:Programfiles\devspace"
wget -UseBasicParsing ((Invoke-WebRequest -URI "https://api.github.com/repos/covexo/devspace/releases/latest" -UseBasicParsing).Content -replace ".*`"(https://github.com[^`"]*devspace-windows-amd64.exe)`".*","`$1") -o $Env:Programfiles\devspace\devspace.exe
& "$Env:Programfiles\devspace\devspace.exe" "install"
```

</Details>

<Details>
<Summary><b>via Mac Terminal</b></Summary>

```
curl -s -H "Accept: application/json" "https://api.github.com/repos/covexo/devspace/releases/latest" | sed -nE 's!.*"(https://github.com[^"]*devspace-darwin-amd64)".*!\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace
sudo mv devspace /usr/local/bin
```

</Details>

<Details>
<Summary><b>via Linux Bash</b></Summary>

```
curl -s -H "Accept: application/json" "https://api.github.com/repos/covexo/devspace/releases/latest" | sed -nE 's!.*"(https://github.com[^"]*devspace-linux-amd64)".*!\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace
sudo mv devspace /usr/local/bin
```

</Details>

### 2. Install Docker

DevSpace uses Docker to build container images, so you need Docker on your local computer. If you do not have Docker installed yet, you can download the latest stable releases here:
- **Mac**: [Docker Community Edition](https://download.docker.com/mac/stable/Docker.dmg)
- **Windows Pro**: [Docker Community Edition](https://download.docker.com/win/stable/Docker%20for%20Windows%20Installer.exe)
- **Windows 10 Home**: [Docker Toolbox](https://download.docker.com/win/stable/DockerToolbox.exe) (legacy)


### 3. Containerize your application
Run this command within your project:
```
devspace init
```
<Details>
<Summary><b>Don't have a project to test DevSpace with?</b> Check out our example project.</Summary>

```
TODO
```

</Details>

<br>

**What does `devspace init` do?**  
DevSpace.cli will automatically detect your programming language and ask for the ports your application is listening on. It will then, create an Helm chart and a Dockerfile within your project, if you do not already have one.

### 4. Create a Space
This command will create and configure a Kubernetes namespace for you:
```
devspace create space my-app
```
> If you are **not** using DevSpace.cloud, you will not be able to create Spaces. You can skip this step and deploy your application to a regular Kubernetes namespace using `devspace deploy`.

### 5. Deploy your application
Deploy your application to your newly created Space:
```
devspace deploy my-app
```

### What's next?
 - [Connecting custom domains](#)
 - [Debugging deployments with DevSpace](#)
 - [Developing applications with DevSpace](#)

<br>

## Architecture

<p align="center"><a href="#"><img src="docs/website/static/img/github-readme-architecture.gif" alt="DevSpace Architecture" width="100%"></a></p>

<table width="100%">
    <tbody width="100%">
        <tr>
            <td width="25%"></td>
            <td width="25%" align="center"><b>Hosted Spaces</b></td>
            <td width="25%" align="center"><b>Self-hosted Spaces</b></td>
            <td width="25%" align="center"><b>Self-managed Namespaces</b></td>
        </tr>
        <tr>
            <td align="right">DevSpace.cli</td>
            <td align="center"><b>✓</b></td>
            <td align="center"><b>✓</b></td>
            <td align="center"><b>✓</b></td>
        </tr>
        <tr>
            <td align="right">managed with DevSpace.cloud</td>
            <td align="center"><b>✓</b></td>
            <td align="center"><b>✓</b></td>
            <td align="center"><b>✗</b></td>
        </tr>
        <tr>
            <td align="right">hosted on DevSpace.host</td>
            <td width="25%" align="center"><b>✓</b></td>
            <td width="25%" align="center"><b>✗</b></td>
            <td width="25%" align="center"><b>✗</b></td>
        </tr>
        <tr>
            <td align="right">What management effort do I have?</td>
            <td width="25%" align="center">
                <b>No admin effort.</b><br>
                <small>DevSpace.cloud manages everything.</small>
            </td>
            <td width="25%" align="center">
                <b>You create and connect the cluster.</b><br>
                <small>DevSpace.cloud manages users and Spaces on top of it.</small>
            </td>
            <td width="25%" align="center">
                <b>You manage everything.</b><br>
                <small>You manually create, secure and maintain cluster, users and namespaces</small>
            </td>
        </tr>
    </tbody>
</table>

Because you can fairly easy [switch between the three modes uf using DevSpace](#), it generally makes sense to start with **Hosted Spaces** and switch to one of the other modes later on.

<Details>
<Summary><b>DevSpace.cli</b> • swiss army knife for Kubernetes</Summary>

DevSpace.cli is an open-source command-line tool that provides everything you need to develop, deploy and debug applications with Docker and Kubernetes.

> You can either use DevSpace.cli as standalone solution for your self-managed Kubernetes namespaces or in combination with DevSpace.cloud.

</Details>

<Details>
<Summary><b>DevSpace.cloud</b> • management platform for Spaces</Summary>

DevSpace.cloud is a developer platform for Kubernetes that lets you create and manage Spaces via DevSpace.cli or GUI. 

> The Spaces you create with DevSpace.cloud either run on DevSpace.host or on your own Kubernetes clusters after connecting them to the platform.

</Details>

<Details>
<Summary><b>DevSpace.host</b> • hosting service for Spaces</Summary>

DevSpace.host is a hosting service that lets you create Spaces instead of entire Kubernetes clusters. Because you only pay for the resources used for creating your containers, it is much cheaper than having to pay for an entire Kubernetes cluster, especially for small and medium size workloads.

> DevSpace.host is runs on top of Google Cloud, AWS and Azure clusters and is optimized for reliability and scalability.

</Details>


<br>

## Contributing
Help us make DevSpace.cli the best tool for developing, deploying and debugging Kubernetes apps.

### Reporting Issues
If you find a bug while working with the DevSpace.cli, please [open an issue on GitHub](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

### Feedback & Feature Requests
You are more than welcome to open issues in this project to:
- [give feedback](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Ffeedback&title=Feedback:)
- [suggest new features](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Ffeature&template=feature-request.md&title=Feature%20Request:)
- [ask a question on Slack](https://devspace.cloud/slack)

### Contributing Code
This project is mainly written in Golang. If you want to contribute code:
1. Ensure you are running golang version 1.11.4 or greater for go module support.
2. Check-out the project: `git clone https://github.com/devspace-cloud/devspace && cd devspace`
3. Make changes to the code (dependencies are downloaded when you run any go command such as `go build`)
4. Build the project, e.g. via `go build -o devspace.exe`

See [Contributing Guideslines](CONTRIBUTING.md) for more information.


<br>

## FAQ
<Details>
<Summary>Do I need a Kubernetes cluster to use DevSpace?</Summary>

**No.** You can simply use **Hosted Spaces** which run on top of DevSpace.host and which are fully managed by DevSpace.cloud.

</Details>

<Details>
<Summary>Can I use DevSpace with my existing Kubernetes clusters?</Summary>

**Yes.** You have two options:
1. [Connect your existing Kubernetes clusters to DevSpace.cloud](#) as external clusters. DevSpace.cloud will then be able to create and manage users and Spaces on opt of your Kubernetes clusters.
2. You just use DevSpace.cli without DevSpace.cloud. That means that you manually need to:
    * enforce resource limits
    * configure secure user permissions
    * isolate namespaces of different users
    * connect domains and configure ingresses
    * install and manage basic cluster services (e.g. ingress controller, cert-manager for TLS, monitoring and log aggregation tools)

</Details>

<Details>
<Summary>Do I need to be a Kubernetes expert to use DevSpace?</Summary>

**No.** Altough DevSpace provides a lot of advanced tooling for Kubernetes experts, it is optimized for developer experience which makes it especially easy to use for Kubernetes beginners.

</Details>

<Details>
<Summary>What is a Space?</Summary>

Spaces are smart Kubernetes namespaces which provide the following features:
- Automatic provisioning via `devspace create space [SPACE_NAME]`
- Automatic allocation of a subdomain for each Space, e.g. `my-app.devspace.host`
- Automatic RBAC configuration for better isolation of users
- Automatic resource limit configuration and enforcement
- Resource auto-scaling within the configured limits
- Smart analysis of issues within your Space via `devspace analyze`

</Details>

<Details>
<Summary>What is DevSpace.cli?</Summary>

DevSpace.cli is an open-source command-line tool that provides everything you need to develop, deploy and debug applications with Docker and Kubernetes.

> You can either use DevSpace.cli as standalone solution for your self-managed Kubernetes namespaces or in combination with DevSpace.cloud.

</Details>

<Details>
<Summary>What is DevSpace.cloud?</Summary>

DevSpace.cloud is a developer platform for Kubernetes that lets you create and manage Spaces via DevSpace.cli or GUI. 

> The Spaces you create with DevSpace.cloud either run on DevSpace.host or on your own Kubernetes clusters after connecting them to the platform.

</Details>

<Details>
<Summary>What is DevSpace.host?</Summary>

DevSpace.host is a hosting service that lets you create Spaces instead of entire Kubernetes clusters. Because you only pay for the resources used for creating your containers, it is much cheaper than having to pay for an entire Kubernetes cluster, especially for small and medium size workloads.

> DevSpace.host is runs on top of Google Cloud, AWS and Azure clusters and is optimized for reliability and scalability.

</Details>

<Details>
<Summary>What is a Helm chart?</Summary>

[Helm](#) is the package manager for Kubernetes. Packages in Helm are called Helm charts.

[Learn more about Helm charts](#)

</Details>


<br>

## License
You can use the DevSpace.cli for any private or commercial projects because it is licensed under the Apache 2.0 open source license.
