---
title: Workflow & basics
---

With DevSpace CLI, you can automate the manual work of building, tagging and pushing Docker images. Simply [define an image in your DevSpace configuration](/docs/cli/deployment/images) and DevSpace CLI will:

1. Build a new image if the Dockerfile or the Docker context has changed
2. Apply [entrypoint overrides](/docs/cli/development/entrypoint-overrides) for development (only when running `devspace dev`)
3. Tag this new image with an auto-generated tag
4. Push this image to any [Docker registry](/docs/cli/images/workflow) of your choice

When running `devspace deploy` or `devspace dev`, DevSpace CLI will continue with deploying your application as defined in the `deployments`. Before deploying, DevSpace CLI will use the newly generated tag and replace every occurence of the same image in your deployment files (e.g. Helm charts or Kubernetes manifests) with the newly generated tag, so that you are always deploying the newest version of your application. This tag replacement happens entirely in-memory, so your deployment files will not be altered.

To make sure that Kubernetes can pull your image even when you are pushing to a private registry (such as dscr.io), DevSpace CLI will also create an [image pull secret](/docs/cli/images/pull-secrets) containing credentials for your registry.

> If you have multiple Dockerfiles in your project (e.g. in case of a monorepo), you can also tell DevSpace CLI to build multiple images in a row.
