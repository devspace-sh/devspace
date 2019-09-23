---
title: Workflow & Basics
id: version-v4.0.1-workflow-basics
original_id: workflow-basics
---

DevSpace lets you build applications directly inside a Kubernetes cluster with this command:
```bash
devspace dev
```

<br>
<img src="/img/processes/image-building-process-devspace.svg" alt="DevSpace Image Building Process" style="width: 100%;">


## Advantages
The biggest advantages of developing directly inside Kubernetes is that your dev environment will be very similar to your production environment and you can have a much greater confidence that everything will work in production when shipping new features.

Kubernetes-based development can be useful in the following cases:
- Your applications needs to access cluster-internal services (e.g. Cluster DNS)
- You want to test your application in a production-like environment
- You want to debug issues that are hard to reproduce on your local machine

The development experience is very similar to using `docker-compose`, so if you are already familiar on how to develop with `docker-compose`, DevSpace will behave very similar. One of the major benefits of DevSpace versus docker-compose is that DevSpace allows you to develop in any Kubernetes cluster, either locally using minikube, Docker Kubernetes etc. or in any remote Kubernetes cluster.  


## Start Development Mode
Start the development mode using this command:
```bash
devspace dev
```

> **It is highly discouraged to run `devspace dev` multiple times in parallel** because multiple instances of port-forwarding and code synchronization will disturb each other. [Run `devspace enter` to open additional terminals](#devspace-enter) without port-forwarding and code synchronization.

### Important Flags for `devspace dev`
The following flags are available for all commands that trigger image building:
- `-b / --force-build` rebuild all images (even if they could be skipped because context and Dockerfile have not changed)
- `-d / --force-deploy` redeploy all deployments (even if they could be skipped because they have not changed)
- `-i / --interactive` starts the [interactive mode](/docs/cli/development/configuration/interactive-mode)



## Development Process
The development process first runs the [deployment process](/docs/cli/deployment/workflow-basics) (1. - 4.) and then continues with starting the development-specific features.

### 1. Build &amp; Deploy Dependencies
DevSpace loads the `dependencies` section from the `devspace.yaml` and creates a dependency tree. The current project will represent the root of this tree. Based on this dependency tree, DevSpace will start from the leaves and run these steps for each dependency:
- Build images of the dependency as configured in the `images` section of the dependency's `devspace.yaml` (unless `skipBuild: true`)
- Deploy the dependency as configured in the `deployments` section of the dependency's `devspace.yaml`

[Learn more about deploying dependencies with DevSpace.](/docs/cli/deployment/advanced/dependencies)

> Dependencies allow you to deploy microservices, that the project you are currently deploying relies on. Dependencies can be located in a subpath of your project or they can be automatically loaded from a different git reporsitory.


### 2. Build, Tag &amp; Push Images
DevSpace triggers the [image building process](/docs/cli/image-building/workflow-basics) for the images specified in the `images` section of the `devspace.yaml`.

[Learn more about image building with DevSpace.](/docs/cli/image-building/workflow-basics)


### 3. Tag Replacement
After finishing the image building process, DevSpace searches your deployments for references to the images that are specified in the `images` section of the `devspace.yaml`. If DevSpace finds that an image is used by one of your deployments and the deployment does not explicitly define a tag for the image, DevSpace will append the tag that has been auto-generated as part of the [automated image tagging](/docs/cli/image-building/workflow-basics#6-tag-image) during the image building process.

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


### 5. Start Port-Forwarding
DevSpace iterates over every item in the `dev.ports` array defined in the `devspace.yaml` and starts port-forwarding for each of the entries and the port mappings they define in the `forward` section.

Before starting the actual port-forwarding threads, DevSpace waits until the containers and services are ready.

> Port-Fowarding allows you to access your containers and Kubernetes services via localhost.

For detailed logs about the port-forwarding, take a look at `.devspace/logs/portforwarding.log`.

### 6. Start File Synchronization
DevSpace iterates over every item in the `dev.sync` array defined in the `devspace.yaml` and starts a bi-directional, real-time code synchronization for each of the entries and the path mappings they define.

Right after starting the file synchronization, DevSpace runs the so-called initial sync which quickly computes the differences between your local folders and the remote container filesystems. If DevSpace detects changes, it synchronizes them first to get a clean state before starting the real-time synchronication which gets invokes every time a file changes.

<details>
<summary>

#### How Does The File Synchronization Work?

</summary>

Before starting the actual file synchronization, DevSpace waits until the containers are up and running and injects a sync binary directly inside the containers. To start the file synchronization, DevSpace starts this binary inside the container and connects to this process through the terminal. This procedure is very lightweight and allows DevSpace to make the file synchronization much more reliable, fast and secure than with any other tool available.

> You can see the file synchronization processes running in your container by running `ps aux` inside the container.

</details>

For detailed logs about the file synchronzation, take a look at `.devspace/logs/sync.log` for the current session and `.devspace/logs/sync.log.old` for previous logs.


### 7. Start Log Streaming (or Interactive Terminal)
DevSpace provides two options to develop applications in Kubernetes:
- using multi-container log streaming (default)
- using an interactive terminal session (run `devspace dev -i`)

#### Multi-Container Log Streaming (Option A, default)
The first option starts your application as defined in your Dockerfile or in your Kubernetes pod definition. After the pods are started, DevSpace streams the logs of all containers that are started with an image that was built during the [image building process](/docs/cli/image-building/workflow-basics). Each log line is prefixed with the image name or alternatively with the pod name of the container. Before starting the actual log streaming, DevSpace prints the last 50 log lines of each container by default.

Learn how to [customize which containers should be included in the log stream and how many log lines should be shown in the beginning](/docs/cli/development/configuration/logs-streaming).

#### Interactive Terminal Session (Option B)
To start interactive mode, run:
```bash
devspace dev -i
```
Instead of starting the multi-container log streaming, you can also start development mode using an interactive terminal session. This interactive mode builds your images (by default) using an `ENTRYPOINT = [sleep, 999999]` override for the image you want to work on and starts an interactive terminal session for the container that is being started with this image. This means that your container starts but without starting your application which allows you to run a command through the terminal session to manually start the application. This is often useful for debugging container start issues or for quickly testing different commands that could be used as an `ENTRYPOINT`.

Interactive mode works out of the box but is also [customizable using the `dev.interactive` configuration section](/docs/cli/development/configuration/interactive-mode).

### 8. Open The Browser (optional)
DevSpace iterates over every item in the `dev.open` array defined in the `devspace.yaml` and tries to open the URL you provide for each item using the following method:

1. DevSpace starts to periodically send `HTTP GET` requests to the provideded `dev.open[*].url`.
2. As soon as the first HTTP response has a status code which is neither 502 (Bad Gateway) nor 503 (Service Unavailable), DevSpace assumes that the application is now started, stops sending any further requests and opens the provided URL in the browser.
3. If the URL is still returning status code 502 or 503 after 4min, DevSpace will stop trying to open it. To not disturb the log streaming or the interactive terminal session, DevSpace will not show an error when hitting the 4min timeout.

Learn more about [configuring auto-open](/docs/cli/development/configuration/auto-open).


## Useful Commands

### `devspace dev -i`
To start development in interactive mode, run:
```bash
devspace dev -i
```

Learn more about [starting development using interactive mode](#interactive-terminal-session-option-b).

### `devspace enter`
The command `devspace dev -i` starts a terminal but it also starts port-forwarding and file synchronization which can only be opened once. However, often you need additional terminal sessions. To open a simple terminal session without starting port-forwarding and file sync, run the following command:
```bash
devspace enter
```

If you do not provide a selector (e.g. pod name, label selector or image selector), DevSpace will show a picker with all available pods and containers.

> This command is a general purpose command which also works for any pod/container in Kubernetes even if you are not within a DevSpace project.

### `devspace logs [-f]`
If you want to print or stream the logs of a single container, run:
```bash
# Print logs
devspace logs

# Stream logs
devspace logs -f
```

If you do not provide a selector (e.g. pod name, label selector or image selector), DevSpace will show a picker with all available pods and containers.

> This command is a general purpose command which also works for any pod/container in Kubernetes even if you are not within a DevSpace project.

### `devspace sync`
If you want to start code synchronization on-demand (and even outside a DevSpace project), you can run commands like the ones shown here:
```bash
devspace sync --local-path=subfolder --container-path=/app
devspace sync --exclude=node_modules --exclude=test
devspace sync --pod=my-pod --container=my-container
```

If you do not provide a selector (e.g. pod name, label selector or image selector), DevSpace will show a picker with all available pods and containers.

> This command is a general purpose command which also works for any pod/container in Kubernetes even if you are not within a DevSpace project.

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
To use the second option, you either need to make sure the DNS of your domain points to your Kubernetes cluster and you have an ingress-controller running in your cluster OR you use [DevSpace Cloud](/docs/cloud/what-is-devspace-cloud), either in form of Hosted Spaces or by connecting your own cluster using the command `devspace connect cluster`.

> If your application does not open as exepected, run [`devspace analyze` and DevSpace will try to identify the issue](#devspace-analyze).

### `devspace analyze`
If your application is not starting as expected or there seems to be some kind of networking issue, you can let DevSpace run an automated analysis of your namespace using the following command:
```bash
devspace analyze
```
After analyzing your namespace, DevSpace compiles a report with potential issues, which is a good starting point for debugging and fixing issues with your deployments.

### `devspace list commands`
DevSpace allows you to share commands for common development tasks which can be executed with `devspace run [command-name]`. To get a list of available commands, run:
```bash
devspace list commands
```
Learn how to [configure shared commands for `devspace run`](/docs/cli/development/advanced/shared-commands).

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
