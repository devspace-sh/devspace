---
title: Architecture
---

Architecturally, the DevSpace.cli is a client-side software that interacts with services within your Kubernetes cluster. While the DevSpace.cli can deploy required services (e.g. image registry, Tiller server, Kaniko build pods) automatically, you can also configure it to use already deployed or externally hosted services.

![DevSpace.cli Architecture](/img/devspace-architecture.svg)

The following paragraphs describe the different functions of the DevSpace.cli and how they interact with other system components. 

**Note:** Any interaction between your local computer and your DevSpace is passed through your Kubernetes API server, so you should ensure that your API server is protected with a suitable configuration for using TLS.

## Image Building
Whenever you run `devspace up`, the DevSpace.cli will check if your [Dockerfile](/docs/configuration/dockerfile.html) has changed since the last build. When changes are detected, it re-builds the Docker image. By default, it will use Kaniko to build Docker images directly inside your Kubernetes cluster. Thereby, the DevSpace.cli starts a build pod and runs Kaniko inside it. Kaniko allows to build Docker images without using Docker and completely within the userspace, so you do not need privileged pods or any other special configuration. Soon, the DevSpace.cli will also support building Docker images with a Docker daemon running on your local machine. When using Kaniko, it is not required to have Docker installed locally.

After building the Docker image, the DevSpace.cli will push this newly built image to the specified image registry. By default, the DevSpace.cli deploys a private registry inside your Kubernetes cluster. Alternatively, you can configure to use DockerHub or any other compatible registry.

**Note:** If you do not have a [Dockerfile](/docs/configuration/dockerfile.html) yet, the DevSpace.cli will automatically create one for you.

## Chart Deployment
The DevSpace.cli will deploy the Helm chart specified under [chart/](/docs/configuration/chart.html) inside your source code directory. To deploy the chart, the DevSpace.cli requires a Tiller server to be installed inside your Kubernetes cluster. If you do not have a Tiller server running inside your cluster, the DevSpace.cli will automatically deploy a Tiller server. You do not need to install Helm locally to use the DevSpace.cli.

**Note:** If you do not have a [Helm chart](/docs/configuration/chart.html) yet, the DevSpace.cli will automatically create one for you.

## Terminal Proxy
The terminal proxy opens a terminal within your dev container and connects your local command-line to it. This way, you can directly run commands inside your DevSpace and stream the response directly to your local computer.

## Port Forwarding
With port forwarding, you can access DevSpace-internal ports (e.g. ports of pods and services) via localhost. This is not only useful to access your application with a regular browser using an address such as `localhost:5000` but also to make use of the remote debugging capabilities provided by your IDE.

**Note:** You can configure the the port mappings for port forwarding via [.devspace/config.yaml](/docs/configuration/config.yaml.html).
