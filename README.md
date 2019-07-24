<img src="docs/website/static/img/devspace-logo.svg">

<img src="docs/website/static/img/readme/line.svg" height="1">

### **[Quickstart](#quickstart)** • **[Examples](#configuration-examples)** • **[Documentation](https://devspace.cloud/docs)** • **[Slack](https://devspace.cloud/slack)** &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  [![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/intent/tweet?text=Check%20out%20%23DevSpace%20-%20it%20lets%20you%20build%20cloud-native%20applications%20faster%20and%20automate%20the%20deployment%20process%20to%20%23Kubernetes%20https%3A//github.com/devspace-cloud/devspace/%0A%23cncf%20%23cloudnative%20%23cloud%20%23docker%20%23containers) [![Build Status](https://travis-ci.org/devspace-cloud/devspace.svg?branch=master)](https://travis-ci.org/devspace-cloud/devspace) [![Go Report Card](https://goreportcard.com/badge/github.com/devspace-cloud/devspace)](https://goreportcard.com/report/github.com/devspace-cloud/devspace) [![Slack](https://devspace.cloud/slack/badge.svg)](http://devspace.cloud/slack)

<img src="docs/website/static/img/readme/line.svg" height="1">

### DevSpace makes it easier and faster to build applications for Kubernetes
- **Build, test and debug applications directly inside Kubernetes**
- **Automate repetitive tasks** for image building and deployment
- **Unify deployment workflows** among developers or across dev, staging and production

<br>


[![DevSpace Demo](docs/website/static/img/readme/devspace-cli-demo.gif)](https://youtu.be/0-5XwDeIG0s)

<p align="center">
<a href="https://youtu.be/0-5XwDeIG0s">Click here to watch the full-length video with sound on YouTube [4min]</a><br><br> ⭐️ <strong>Do you like DevSpace? Support the project with a star</strong> ⭐️
</p>


<br>

## Contents
- [Features](#features)
- [Architecture](#architecture)
- [Quickstart](#quickstart)
- [Config Examples](#configuration-examples)
- [Contributing](#contributing)
- [FAQ](#faq)

<br>

## Features

Stop wasting time for running the same build and deploy commands over and over again. Let DevSpace automate your workflow and build cloud-native applications directly inside Kubernetes.

### Automatic Image Building 
  
- **Customizable Build Process** supporting Docker, kaniko or even custom scripts
- **Parallel Image Building** to save time when multiple Dockerfiles have to be built  
- **Automatic Image Tagging** according to custom tag schema (e.g. using timestamp, commit hash or random strings)  
- **Automatic Push** to any public or private Docker registry (authorization via `docker login my-registry.tld`)  
- **Automatic Configuration of Pull Secrets** within the Kubernetes cluster
- **Smart Caching** that skips images which do not need to be rebuilt

### Automatic Deployment with `devspace deploy`
- **Automatig Image Building** for images required in the deployment process
- **Customizable Deployment Process** supporting kubectl, helm, kustomize and more
- **Multi-Step Deployments** to deploy multiple application components (e.g. 1. webserver, 2. database, 3. cache)
- **Efficient Microservice Deployments** by defining dependencies between projects (even across git repositories)
- **Smart Caching** that skips deployments which do not need to be redeployed
- **Easy Integration into CI/CD Tools** with non-interactive mode


### Efficient In-Cluster Development with `devspace dev`
- **Hot Reloading** that updates your running containers without restarting them (whenever you change a line of code)
- **Fast + Reliable File Synchronization** to keep all files in sync between your local workspace and your containers
- **Terminal Proxy** that opens automatically and lets you run commands in your pods directly from your IDE terminal
- **Port Forwarding** that lets you access services and pods on localhost and allows you to attach debuggers with ease


### Faster Interaction with Kubernetes
- **Quick Pod Selection** eliminates the need to copy & paste pod names, namespaces etc.  
  &raquo; Shows a "dropdown selector" for pods directly in the CLI when running one of these commands:
  - `devspace enter` to open a terminal session **Fast, Real-Time Log Streaming** for all containers you deploy
  - `devspace logs` / `devspace logs -f` for **Fast, Real-Time Logs** (optionally streaming new logs)
  - `devspace sync` for quickly starting a **Bi-Directional, Real-Time File Synchronization** on demand 
- **Automatic Issue Analysis** via `devspace analyze` reporting crashed containers, missing endpoints, scheduling errors, ...
- **Fast Deletion of Deployments** using `devspace purge` (deletes all helm charts, manifests etc. defined in the config)


### Powerful Configuration
- **Declarative Configuration File** that can be versioned and shared just like the source code of your project (e.g. via git)
- **Config Variables** which allow you to parameterize the config and share a unified config file with your team
- **Config Overrides** for overriding Dockerfiles or ENTRPOINTs (e.g. to separate development, staging and production)
- **Hooks** for executing custom commands before or after each build and deployment step
- **Multiple Configs** for advanced deployment scenarios


### Lightweight & Easy to Setup
- **Client-Only Binary** (server-side DevSpace Cloud is optional for visual UI and team management, see [Architecture](#architecture))
- **Standalone Executable for all platforms** with no external dependencies and *fully written in Golang*
- **Automatic Config Generation** from existing Dockerfiles, Helm chart or Kubernetes manifests (optional)
- **Automatic Dockerfile Generation** (optional)

### Management UI for Teams & Dev Clusters *(optional, using [DevSpace Cloud](https://github.com/devspace-cloud/devspace-cloud))*
- **Graphical UI** for managing clusters, cluster users and user permissions (resource limits etc.)
- **On-Demand Namespace Creation & Isolation** with automatic RBAC, network policies, pod security policies etc.
- **Advanced Permission System** that automatically enforces user limits via resource quotas, adminission controllers etc.
- **Fully Automatic Context Configuration** on the machines of all cluster users with secure access token handling
- **100% Pure Kubernetes** and nothing else! Works with any Kubernetes cluster.

**More info and install intructions for DevSpace Cloud on: [www.github.com/devspace-cloud/devspace-cloud](https://github.com/devspace-cloud/devspace-cloud)**


<br>


## Architecture
![DevSpace Architecture](docs/website/static/img/readme/devspace-architecture.png)

DevSpace runs as a single binary CLI tool directly on your computer and ideally, you use it straight from the terminal within your IDE. DevSpace does not require a server-side component as it communicates directly to your Kubernetes cluster using your kubectl context.

You can, however, connect your Kubernetes cluster to DevSpace Cloud to manage cluster users, namespaces and permissions with a central management UI. DevSpace Cloud can either be used [as-a-Service on devspace.cloud](https://devspace.cloud) or installed as an on-premise version (see [www.github.com/devspace-cloud/devspace-cloud](https://github.com/devspace-cloud/devspace-cloud) for instructions).

<br>

## Quickstart

### 1. Install

<details>
<summary>via NPM</summary>

```
npm install -g devspace
```

</details>

<details>
<summary>via Mac Terminal</summary>

```
curl -s -L "https://github.com/devspace-cloud/devspace/releases/latest" | sed -nE 's!.*"([^"]*devspace-darwin-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace;
sudo mv devspace /usr/local/bin;
```

</details>

<details>
<summary>via Linux Bash</summary>

```
curl -s -L "https://github.com/devspace-cloud/devspace/releases/latest" | sed -nE 's!.*"([^"]*devspace-linux-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o devspace && chmod +x devspace;
sudo mv devspace /usr/local/bin;
```

</details>

<details>
<summary>via Windows Powershell</summary>

```
md -Force "$Env:APPDATA\devspace"; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]'Tls,Tls11,Tls12';
wget -UseBasicParsing ((Invoke-WebRequest -URI "https://github.com/devspace-cloud/devspace/releases/latest" -UseBasicParsing).Content -replace "(?ms).*`"([^`"]*devspace-windows-amd64.exe)`".*","https://github.com/`$1") -o $Env:APPDATA\devspace\devspace.exe; & "$Env:APPDATA\devspace\devspace.exe" "install"; $env:Path = (Get-ItemProperty -Path HKCU:\Environment -Name Path).Path
```

</details>

<br>

### 2. Choose a Project

| Project | Command                                                                                 |
| ------- | --------------------------------------------------------------------------------------- |
| Node.js | `git clone https://github.com/devspace-cloud/quickstart-nodejs && cd quickstart-nodejs` |
| Python  | `git clone https://github.com/devspace-cloud/quickstart-python && cd quickstart-python` |
| Golang  | `git clone https://github.com/devspace-cloud/quickstart-golang && cd quickstart-golang` |
| PHP     | `git clone https://github.com/devspace-cloud/quickstart-php && cd quickstart-php`       |
| Ruby    | `git clone https://github.com/devspace-cloud/quickstart-ruby && cd quickstart-ruby`     |


<details>
<summary>Want to use DevSpace with your own project?</summary>

```bash
cd /path/to/my/project/root
```

> If you are using DevSpace for the first time, we recommend to get started with one of the demo projects listed above.

</details>


<br>

### 3. Deploy

Choose the cluster, you want to deploy your project to. If you are not sure, pick the first option. It is fairly easy to switch between the options listed here.


<details>
<summary><b>DevSpace Cloud (fully managed clusters, SaaS version)</b>
<br>&nbsp;&nbsp;&nbsp;
<i>
<u>free</u> for one project, includes 1 GB RAM
</i>
</summary>

<br>

```bash
devspace init
devspace create space my-app
devspace deploy
```

</details>

<br>

<details>
<summary><b>DevSpace Cloud (connect your own cluster, SaaS version)</b> 
<br>&nbsp;&nbsp;&nbsp;
for clusters with public IP address, e.g. Google Cloud, AWS, Azure
</summary>

<br>

```bash
devspace connect cluster
devspace init
devspace create space my-app
devspace deploy
```

</details>

<br>

<details>
<summary><b>DevSpace Cloud (self-hosted version)</b> 
<br>&nbsp;&nbsp;&nbsp;
for clusters <u>without</u> public IP address, e.g. bare metal clusters within a VPN
</summary>

<br>

**1. Install DevSpace Cloud**  
&nbsp;&nbsp;&nbsp;
See [www.github.com/devspace-cloud/devspace-cloud](https://github.com/devspace-cloud/devspace-cloud) for instructions.

**2. Tell DevSpace to use your self-hosted DevSpace Cloud**  
```bash
devspace use provider devspace.my-domain.com
```

**3. Connect a Kubernetes cluster to your self-hosted DevSpace Cloud**  
```bash
devspace connect cluster
```

**4. Deploy your project**  
```bash
devspace init
devspace create space my-app
devspace deploy
```

</details>

<br>

<details>
<summary><b>Use current kubectl context (without DevSpace Cloud)</b>
<br>&nbsp;&nbsp;&nbsp;
for local clusters, e.g. minikube, kind, Docker Kubernetes
</summary>

<br>

```bash
devspace init
devspace deploy
```

</details>

<br>

### 4. Develop
After successfully deploying your project one, you can start it in development mode and directly code within your Kubernetes cluster using terminal proxy, port forwarding and real-time code synchronization.

```bash
devspace dev
```
DevSpace will deploy your application, wait until your pods are ready and open the terminal of a pod that is specified in your config. You can now start your application manually using a command such as `npm start` or `npm run develop` and access your application via `localhost:PORT` in your browser. Edit your source code files and DevSpace will automatically synchronize them to the containers in your Kubernetes cluster.

> If you are using DevSpace Cloud, you can now `devspace ui` to open the graphical user interface in your browser, stream logs, add new users to your cluster and configure permissions for everyone on your team.

<br>

### 5. Learn more

<details>
<summary>Show useful commands for development
</summary>

<br>

| Command&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; | Important flags                                                                                      |
| ------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `devspace dev`<br> Starts the development mode                      | `-b • Rebuild images (force)` <br> `-d • Redeploy everything (force)`                                |
| `devspace enter`<br> Opens a terminal session for a container       | `-p • Pick a container instead of using the default one`                                             |
| `devspace enter [command]`<br> Runs a command inside a container    |                                                                                                      |
| `devspace logs` <br> Prints the logs of a container                 | `-p • Pick a container instead of using the default one` <br> `-f • Stream new logs (follow/attach)` |
| `devspace analyze` <br> Analyzes your deployments for issues        |                                                                                                      |

</details>

<details>
<summary>Show how to use multiple Spaces and switch between them
</summary>

Whenever you run `devspace create space [space-name]` within a project, DevSpace will set this newly created Space as active Space which is used for `devspace deploy` and `devspace dev`.

If you have multiple Spaces (e.g. to test different versions of your app or separate staging from production), you can use the following commands to list Spaces and switch between them:

```bash
devspace list spaces
devspace use space [space-name]
```

</details>


<br>
<br>


## Configuration Examples
You can configure DevSpace with the `devspace.yaml` configuration file that should be placed within the root directory of your project. The general structure of a `devspace.yaml` looks like this:

```yaml
version: {config-version}

images:                 # DevSpace will build these images in parallel and push them to the respective registries
  {image-a}: ...        # tells DevSpace how to build image-a
  {image-b}: ...        # tells DevSpace how to build image-b
  ... 

deployments:            # DevSpace will deploy these [Helm chart | manifest | ... ] after another
  - {deployment-1}      # could be a Helm chart
  - {deployment-2}      # could be a folder with kubectl manifests
  ...

dev:                    # Special config options for `devspace dev`
  overrideImages: ...   # Apply overrides to image building (e.g. different Dockerfile or different ENTRYPOINT)
  terminal: ...         # Config options for opening a terminal or streaming logs
  ports: ...            # Config options for port-forwarding
  sync: ...             # Config options for file synchronization
  autoReload: ...       # Tells DevSpace when to redeploy (e.g. when a manifest file has been edited)

dependencies:           # Tells DevSpace which related projects should be deployed before deploying this project
  - {dependency-1}      # Could be another git repository
  - {dependency-2}      # Could point to a path on the local filesystem
  ...
```

<details>
<summary>Show me an example of a devspace.yaml config file</summary>

```yaml
version: v1beta2

images:
  default:                              # Key 'default' = Name of this image
    image: my-registry.tld/image1       # Registry and image name for pushing the image
    createPullSecret: true              # Let DevSpace CLI automatically create pull secrets in your Kubernetes namespace

deployments:
- name: quickstart-nodejs               # Name of this deployment
  component:                            # Deploy a component (alternatives: helm, kubectl)
    containers:                         # Defines an array of containers that run in the same pods started by this component
    - image: my-registry.tld/image1     # Image of this container
      resources:
        limits:
          cpu: "400m"                   # CPU limit for this container
          memory: "500Mi"               # Memory/RAM limit for this container
    service:                            # Expose this component with a Kubernetes service
      ports:                            # Array of container ports to expose through the service
      - port: 3000                      # Exposes container port 3000 on service port 3000

dev:
  overrideImages:
  - name: default
    entrypoint:
    - sleep
    - 9999999
  ports:
    forward:
    - port: 8080
      remotePort: 80
    - port: 3000
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
  sync:
  - localSubPath: ./src
    containerPath: .
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
  autoReload:
    paths:
    - ./manifests/**

dependencies:
- source:
    git: https://github.com/my-api-server
- source:
    path: ../my-auth-server
```

</details>

<br>

The following sections show code snippets with example sections of a `devspace.yaml` for certain use cases. 


### Configure Image Building

<details>
<summary>
Build images with Docker
</summary>

```yaml
images:
  auth-server:
    image: dockerhub-username/my-auth-server    # Push to Docker Hub (no registry hostname required) => uses ./Dockerfile by default
    createPullSecret: true                      # Create a Kubernetes pull secret for this image before deploying anything
  webserver:
    image: dscr.io/username/my-webserver        # Push to private registry
    createPullSecret: true
    dockerfile: ./webserver/Dockerfile          # Build with --dockerfile=./webserver/Dockerfile
    context: ./webserver                        # Build with --context=./webserver
  database:
    image: another-registry.tld/my-image        # Push to another private registry
    createPullSecret: true
    dockerfile: ./db/Dockerfile                 # Build with --dockerfile=./db/Dockerfile
    context: ./db                               # Build with --context=./db
    # The following line defines a custom tag schema for this image (default tag schema is: ${DEVSPACE_RANDOM})
    tag: ${DEVSPACE_USERNAME}-devspace-${DEVSPACE_GIT_COMMIT}-${DEVSPACE_RANDOM}
```
Take a look at the documentation for more information about [configuring builds with Docker](https://devspace.cloud/docs/image-building/build-tools/docker).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Build images with kaniko (inside a Kubernetes pod)
</summary>

```yaml
images:
  auth-server:
    image: dockerhub-username/my-auth-server    # Push to Docker Hub (no registry hostname required) => uses ./Dockerfile by default
    build:
      kaniko:                                   # Build this image with kaniko
        cache: true                             # Enable caching
        insecure: false                         # Allow kaniko to push to an insecure registry (e.g. self-signed SSL certificate)
  webserver:
    image: dscr.io/username/my-webserver        # This image will be built using Docker with kaniko as fallback if Docker is not running
    createPullSecret: true
    dockerfile: ./webserver/Dockerfile          # Build with --dockerfile=./webserver/Dockerfile
    context: ./webserver                        # Build with --context=./webserver
```
Take a look at the documentation for more information about [building images with kaniko](https://devspace.cloud/docs/image-building/build-tools/kaniko). <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Build images with custom commands and scripts
</summary>

```yaml
images:
  auth-server:
    image: dockerhub-username/my-auth-server    # Push to Docker Hub (no registry hostname required) => uses ./Dockerfile by default
    build:
      custom:
        command: "./scripts/builder"
        args: ["--some-flag", "flag-value"]
        imageFlag: "image"
        onChange: ["./Dockerfile"]
  webserver:
    image: dscr.io/username/my-webserver        # This image will be built using Docker with kaniko as fallback if Docker is not running
    createPullSecret: true
    dockerfile: ./webserver/Dockerfile          # Build with --dockerfile=./webserver/Dockerfile
    context: ./webserver                        # Build with --context=./webserver
```
Take a look at the documentation for more information about using [custom build scripts](https://devspace.cloud/docs/image-building/build-tools/custom-build-script).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>


### Configure Deployments

<details>
<summary>
Deploy components
</summary>

```yaml
# File: ./devspace.yaml
deployments:
- name: quickstart-nodejs
  component:
    containers:
    - image: my-registry.tld/image1
      resources:
        limits:
          cpu: "400m"
          memory: "500Mi"
```
DevSpace allows you to [add predefined components](https://devspace.cloud/docs/deployment/components/add-predefined-components) using the `devspace add component [component-name]` command. 

Learn more about:
- [What are components?](https://devspace.cloud/docs/deployment/components/what-are-components)
- [Configuration Specification for Components](https://devspace.cloud/docs/deployment/components/specification) 

 <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Deploy helm charts
</summary>

```yaml
# File: ./devspace.yaml
deployments:
- name: default
  helm:
    chart:
      name: redis
      version: "6.1.4"
      repo: https://kubernetes-charts.storage.googleapis.com
```
Learn more about:
- [What are Helm charts?](https://devspace.cloud/docs/deployment/helm-charts/what-are-helm-charts)
- [Configure Helm chart deployments](https://devspace.cloud/docs/deployment/helm-charts/add-charts)

 <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Deploy manifests with kubectl
</summary>

```yaml
# File: ./devspace.yaml
deployments:
- name: my-deployment
  kubectl:
    manifests:
    - my-manifests/
    - more-manifests/
    kustomize: true
```
Take a look at the documentation for more information about [deploying manifests with kustomize](https://devspace.cloud/docs/deployment/kubernetes-manifests/kustomize).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Deploy manifests with kustomize
</summary>

```yaml
# File: ./devspace.yaml
deployments:
- name: my-cache
  helm:
    chart:
      name: redis
      version: "6.1.4"
      repo: https://kubernetes-charts.storage.googleapis.com
- name: my-nodejs-app
  kubectl:
    manifests:
    - manifest-folder/
    - some-other-manifest.yaml
```
Learn more about:
- [What are Kubernetes manifests?](https://devspace.cloud/docs/deployment/kubernetes-manifests/what-are-manifests)
- [Configure manifest deployments](https://devspace.cloud/docs/deployment/kubernetes-manifests/configure-manifests)

 <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Define multiple deployments in one project
</summary>

```yaml
# File: ./devspace.yaml
deployments:
- name: devspace-default
  kubectl:
    manifests:
    - manifest-folder/
    - some-other-manifest.yaml
```

DevSpace processes all deployments of a project according to their order in the `devspace.yaml`. You can combine deployments of different types (e.g. Helm charts and manifests).

Take a look at the documentation to learn more about [how DevSpace deploys projects to Kubernetes](https://devspace.cloud/docs/workflow-basics/deployment).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Define dependencies between projects (e.g. to deploy microservices)
</summary>

```yaml
# File: ./devspace.yaml
dependencies:
- source:
    git: https://github.com/my-api-server
- source:
    git: https://myuser:mypass@my-private-git.com/my-auth-server 
- source:
    path: ../my-auth-server
  config: default
```

Before deploying a project, DevSpace resolves all dependencies and builds a dependency tree which will then be deployed in a buttom-up fashion, i.e. the project which you call `devspace deploy` in will be deployed last.

Take a look at the documentation to learn more about [how DevSpace deploys dependencies of projects](https://devspace.cloud/docs/workflow-basics/deployment/dependencies).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>


### Configure Development Mode

<details>
<summary>
Override Dockerfile entrypoint
</summary>

```yaml
# File: ./devspace.yaml
images:
  default:
    image: dscr.io/my-username/my-image
dev:
  overrideImages:
  - name: default
    entrypoint:
    - sleep
    - 9999999
```

When running `devspace dev` instead of `devspace deploy`, DevSpace would override the ENTRYPOINT og the Dockerfile with `[sleep, 9999999]` when building this image.

Take a look at the documentation to learn more about [how DevSpace applies dev overrides](https://devspace.cloud/docs/development/overrides#configuring-a-different-dockerfile-during-devspace-dev).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Use different Dockerfile for development
</summary>

```yaml
# File: ./devspace.yaml
images:
  default:
    image: dscr.io/my-username/my-image
dev:
  overrideImages:
  - name: default
    dockerfile: ./development/Dockerfile.development
    # Optional use different context
    # context: ./development
```

When running `devspace dev` instead of `devspace deploy`, DevSpace would use the dev Dockerfile as configured in the example above.

Take a look at the documentation to learn more about [how DevSpace applies dev overrides](https://devspace.cloud/docs/development/overrides).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Configure code synchronization
</summary>

```yaml
# File: ./devspace.yaml
dev:
  sync:
  - localSubPath: ./src # relative to the devspace.yaml
    # Start syncing to the containers current working directory (You can also use absolute paths)
    containerPath: .
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
    # Only download changes to these paths, but do not upload any changes (.gitignore syntax)
    uploadExcludePaths:
    - node_modules/
    # Only upload changes to these paths, but do not download any changes (.gitignore syntax)
    downloadExcludePaths:
    - /app/tmp
    # Ignore these paths completely during synchronization (.gitignore syntax)
    excludePaths:
    - Dockerfile
    - logs/
```

The above example would configure the sync, so that:
- local path `./src` will be synchronized to the container's working directory `.` (specified in the Dockerfile)
- `./src/node_modules` would **not** be uploaded to the container

Take a look at the documentation to learn more about [configuring file synchronization during development](https://devspace.cloud/docs/development/synchronization).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Stream logs instead of opening the container terminal
</summary>

```yaml
# File: ./devspace.yaml
dev:
  terminal:
    disabled: true
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
```

Streams the logs of the selected container instead of opening an interactive terminal session.

Take a look at the documentation to learn more about [configuring the terminal proxy for development](https://devspace.cloud/docs/development/terminal).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Redeploy instead of synchronizing code
</summary>

```yaml
# File: ./devspace.yaml
dev:
  autoReload:
    paths:
    - ./Dockerfile
    - ./manifests/**
```

This configuration would tell DevSpace to redeploy your project when the Dockerfile changes or any file within `./manifests`.

Take a look at the documentation to learn more about [configuring auto-reloading for development](https://devspace.cloud/docs/development/auto-reloading).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>


### Advanced Configuration

<details>
<summary>
Use config variables
</summary>

```yaml
# File: ./devspace.yaml
images:
  default:
    image: ${DEVSPACE_USERNAME}/image-name
    tag: ${DEVSPACE_GIT_COMMIT}-${DEVSPACE_TIMESTAMP}
```

DevSpace allows you to use certain pre-defined variables to make the configuration more flexible and easier to share with others. Additionally, you can add your own custom variables.

Take a look at the documentation to learn more about [using variables for dynamic configuration](https://devspace.cloud/docs/configuration/variables).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Define multiple configs
</summary>

```yaml
# File: ./devspace-configs.yaml
config1:
  config:
    path: ../devspace.yaml
config2:
  config:
    path: ../devspace.yaml
  overrides:
  - data:
      images:
        database:
          image: dscr.io/my-username/alternative-db-image
config3:
  config:
    path: ../devspace-prod.yaml
```

If you have complex deployment scenarios which are not easily addressable by dev overrides, you can create a file named `devspace-configs.yaml` and configure multiple different configurations for DevSpace. They can all use the same underlying base configuration and simply apply certain overrides to sections of the configuration (e.g. config1 and config2) OR they can be entirely different configuration files (e.g. config1 and config3).

You can tell DevSpace to use a specific config file using thit command: `devspace use config [config-name]`

Take a look at the documentation to learn more about [using multiple config files](https://devspace.cloud/docs/configuration/multiple-configs).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>

<details>
<summary>
Define hooks
</summary>

```yaml
# File: ./devspace.yaml
hooks:
  - command: echo
    args:
      - "before image building"
    when:
      before:
        images: all
```

The command defined in this hook would be executed before building the images defined in the config.

Take a look at the documentation to learn more about [using hooks](https://devspace.cloud/docs/configuration/hooks).  <img src="docs/website/static/img/readme/line.svg" height="1">

</details>


<br>
<br>

## Contributing

Help us make DevSpace the best tool for developing, deploying and debugging Kubernetes apps.

### Reporting Issues

If you find a bug while working with the DevSpace, please [open an issue on GitHub](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Fbug&template=bug-report.md&title=Bug:) and let us know what went wrong. We will try to fix it as quickly as we can.

### Feedback & Feature Requests

You are more than welcome to open issues in this project to:

- [give feedback](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Ffeedback&title=Feedback:)
- [suggest new features](https://github.com/devspace-cloud/devspace/issues/new?labels=kind%2Ffeature&template=feature-request.md&title=Feature%20Request:)
- [ask a question on Slack](https://devspace.cloud/slack)

### Contributing Code

This project is mainly written in Golang. If you want to contribute code:

1. Ensure you are running golang version 1.11.4 or greater for go module support
2. Set the following environment variables:
   ```
   GO111MODULE=on
   GOFLAGS=-mod=vendor
   ```
3. Check-out the project: `git clone https://github.com/devspace-cloud/devspace && cd devspace`
4. Run `go clean -modcache`
5. Run `go mod vendor`
6. Make changes to the code
7. Build the project, e.g. via `go build -o devspace.exe`
8. Evaluate and test your changes `./devspace [SOME_COMMAND]`

See [Contributing Guideslines](CONTRIBUTING.md) for more information.

<br>

## FAQ

<details>
<summary>What is DevSpace?</summary>

DevSpace is an open-source command-line tool that provides everything you need to develop, deploy and debug applications with Docker and Kubernetes. It lets you streamline deployment workflows and share them with your colleagues through a declarative configuration file `devspace.yaml`.

</details>

<details>
<summary>What is DevSpace Cloud?</summary>

DevSpace Cloud extends DevSpace with a server-side component. It is entirely optional and meant for cluster admins that want to enable their developers to create isolated Kubernetes namespaces on-demand within a development cluster. DevSpace Cloud lets you easily manage cluster users, enforce resource limits and make sure developers can share a dev cluster without getting in the way of each other.

> Even when using DevSpace Cloud, DevSpace directly interacts with the Kubernetes cluster, so you code or commands will never go through DevSpace Cloud.

</details>

<details>
<summary>What is a Space?</summary>

Spaces are isolated Kubernetes namespaces which are managed by DevSpace Cloud and which provide the following features:

- Automatic provisioning via `devspace create space [SPACE_NAME]`
- Automatic allocation of a subdomain for each Space, e.g. `my-app.devspace.host`
- Automatic RBAC configuration for better isolation of users
- Automatic resource limit configuration and enforcement
- Dynamic resource auto-scaling within the configured limits

</details>

<details>
<summary>Do I need a Kubernetes cluster to use DevSpace?</summary>

**No.** You can simply use **the fully managed Spaces** provided by the SaaS version of DevSpace Cloud.

</details>

<details>
<summary>Can I use DevSpace with my existing Kubernetes clusters?</summary>

**Yes.** You have multiple options:

1. Use DevSpace with your current kubectl context (not using DevSpace Cloud at all).
2. Using the SaaS version of DevSpace Cloud and connect your existing Kubernetes clusters to DevSpace Cloud as external clusters (available soon). DevSpace Cloud will then be able to automatically manage cluster users and permissions. This lets you created isolated namespaces (Spaces) within your Kubernetes clusters.
3. Run DevSpace Cloud on-premise and connect your Kubernetes cluster to it in the same way you would use the SaaS version of DevSpace Cloud.

</details>

<details>
<summary>What is a Helm chart?</summary>

[Helm](https://helm.sh/) is the package manager for Kubernetes. Packages in Helm are called Helm charts. [Learn more about Helm charts.](https://devspace.cloud/docs/deployment/helm-charts/what-are-helm-charts)

</details>

<br>
<br>

You can use the DevSpace for any private or commercial projects because it is licensed under the Apache 2.0 open source license.
