---
title: Deploy existing images
---

DevSpace CLI lets you easily define Kubernetes deployments for any existing Docker image.

### Add deployments for existing images
If you want to use a Docker image from Docker Hub or any other registry, you can add a custom component to your deployments using this command:
```bash
devspace add deployment [deployment-name] --image="my-registry.tld/my-username/image"
```
Example using Docker Hub: `devspace add deployment database --image="mysql"`

> If you are using a private Docker registry, make sure to [login to this registry](/docs/image-building/authentication).
