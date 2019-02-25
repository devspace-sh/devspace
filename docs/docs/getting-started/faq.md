---
title: Frequently Asked Questions (FAQ)
sidebar_label: FAQ
---

## DevSpace

### What is a DevSpace?
A DevSpace is a remote workspace that enables cloud-native development directly inside a Kubernetes cluster. You can setup a DevSpace for each of your projects and directly connect it with your local workspace through the DevSpace.cli. 

### What is the DevSpace.cli?
The DevSpace.cli lets you connect your local workspace to a DevSpace. It is an open-source, client-only software that provides real-time code sync, port forwarding, terminal tunneling and more, so programming with your DevSpace feels just like working with a local development runtime.

### Is it free and open source?
Yes. The DevSpace.cli is completely free and open source (Apache-2.0 license). You can use it for private and for commercial projects.

### Why do I need a DevSpace?
There are many use cases where a DevSpace has advantages over regular development on localhost, e.g.:
- you are building complex systems (e.g. with a micro-service architecture)
- you need access to Kubernetes-internal services (e.g. Cluster DNS)
- you want to run algorithms on large amounts of data that change frequently
- you want to share central dev systems without having to deal with authentication etc.
- you are annoyed of the increasing heat and the loud noise your fan makes when running computing-intense processes

### Which programming languages are supported?
You can use any programming languages with your DevSpace. Just use a Docker image that provides the right tooling to build, run and debug applications with the programming language you want to work with.

### Can I use my own Dockerfile?
Yes. If you do not have a Dockerfile yet, the DevSpace.cli will create one for you. If you already have one, the DevSpace.cli will simply work with that one.

### Can I use it with Minikube?
Yes. Just make sure your Minikube cluster works correctly (cluster DNS is started, pods can talk to each other and to the internet etc.).

### Can I use it with self-hosted Kubernetes clusters?
Yes. Just make sure your self-hosted Kubernetes cluster works correctly (cluster DNS is started, pods can talk to each other and to the internet etc.).

### Can I use it with Azure, Google Cloud or Amazon Web Services?
Yes. Just make sure your Kubernetes API server is reachable from your computer and you have the right credentials in place (if kubectl works fine on your terminal, then the DevSpace.cli should also work correctly).

### How do I get my code into the DevSpace?
The DevSpace.cli sets up a real-time code sync for you. This sync mechanism is very reliable and fast. It works with ephemeral container storage as well as with any kind of persistent volumes. The code sync is bi-directional, i.e. it synchronizes changes from your laptop to your DevSpace and the other way around.

### What is port forwarding?
Port forwarding allows you to access a DevSpace port via localhost, e.g. you can access localhost:8080 and this request will be forwarded for example to your DevSpace on port 80.

## Containers & Images

### What is a container?
Containers are isolated process spaces, i.e. each container has its own group of processes that are separated from and not visible for other containers.

### What is a pod?
A pod is a set of containers that share the same network stack and IP address in the pod network of a Kubernetes cluster.

### What is an image?
An image is the blueprint for creating a container, i.e. a container is an instantiation or running version of an image. Docker images are build from Dockerfiles.

### What is an image registry?
An image registry stores a set of container images. Users can push and pull images from an image registry. Images are usually versioned within a registry by using tags.

### What is Docker?
Docker is a runtime and a toolkit for running containers.

### What is a Dockerfile?
A Dockerfile describes the steps to build a Docker image.

### What is Kaniko?
Kaniko is a tool for building Docker images from Dockerfiles without using Docker. Kaniko lets you process a Dockerfile inside a pod within Kubernetes and push the resulting image to an image registry. It does not require any special privileges and runs entirely in userspace.

## Kubernetes & Helm

### What is Kubernetes?
Kubernetes is the leading container orchestration tool. A Kubernetes cluster usually contains an API server, a scheduler, a manager-controller, an etcd storage, a network plugin and a DNS plugin. Kubernetes allows you to run containers and complex container-based applications at scale and with reduced administrational overhead.

### What is Minikube?
Minikube is a 1-node Kubernetes cluster for testing and development purposes. You can run a Minikube cluster on your laptop to get started with Kubernetes.

### What is Helm?
Helm is a package manager for Kubernetes. It allows developers to run container-based applications that someone has packaged for others. Helm also keeps track of deployed applications and lets you review, upgrade or remove them with very little effort.

### What is a Helm chart?
A Helm chart is a packaged application that can be run on top of Kubernetes by using the Helm package manager.

### What is Tiller?
Tiller is the server-side component of Helm. It is a deployment that runs inside your Kubernetes cluster and is responsible for handling the commands that the Helm client sends to it, e.g. for installing a Helm chart.
