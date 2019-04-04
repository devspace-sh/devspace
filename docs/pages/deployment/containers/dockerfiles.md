---
title: Build & deploy Dockerfiles
---

DevSpace CLI lets you easily build Dockerfiles and define deployments for the Docker images created from such Dockerfiles.

### Add deployments for existing Dockerfiles
Run one of the following commands to add a custom component to your deployments based on an existing Dockerfile:
```bash
devspace add deployment [deployment-name] --dockerfile=""
devspace add deployment [deployment-name] --dockerfile="" --image="my-registry.tld/[username]/[image]"
```
The difference between the first command and the second one is that the second one specifically defines where the Docker image should be pushed to after building the Dockerfile. In the first command, DevSpace CLI would assume that you want to use the [DevSpace Container Registry](/docs/cloud/images/dscr-io) provided by DevSpace Cloud.

> If you are using a private Docker registry, make sure to [login to this registry](/docs/image-building/authentication).

After adding a new deployment, you need to manually redeploy in order to start the newly added component together with the remainder of your previouly existing deployments.
```bash
devspace deploy
```
