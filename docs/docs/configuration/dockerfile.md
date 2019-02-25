---
title: /Dockerfile
---

A Dockerfile specifies how a build engine (e.g. Docker or kaniko) will assemble a Docker image. You can add files, install dependencies, set environment variables and run any other command. 

When you run `devspace init` or `devspace up`, the DevSpace.cli will add an exemplary Dockerfile for your project if you do not already have one, yet. You can modify the Dockerfile according to your needs. You can also have multiple Dockerfiles within your project and let the DevSpace.cli build and push them accordingly (see [config.yaml](config.ymal.html)).

See the official Docker documentation for **"[Best practices for writing Dockerfiles](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)"**

**Note: You do not need to install Docker, to use the DevSpace.cli.**
