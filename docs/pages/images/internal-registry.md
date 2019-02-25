---
title: Use dscr.io
---

To make it easier for you to get started with Kubernetes, DevSpace.cloud provides a private Docker registry for you. This registry is called DevSpace Container Registry (dscr.io) and allows you to push and pull images to private repositories. 

> Images in dscr.io have the following format: **dscr.io/[USERNAME]/[IMAGE_NAME]:[TAG]**

## Login to dscr.io
The authentication credentials for dscr.io are automatically generated and fully managed by DevSpace.cli. That means DevSpace.cli will automatically retrieve and securely store your credentials when you login to DevSpace.cloud via:
```bash
devspace login
```

## Use dscr.io with Docker CLI
If you have Docker installed, your credentials for dscr.io will be securely stored using the Docker credentials store. This allows you to also manually push and pull images to/from dscr.io using regular commands of the Docker cli.
```bash
docker build -t dscr.io/username/image:v1 .
docker push dscr.io/username/image:v1
```
The commands shown above would, for example, build a Docker image with your local Docker daemon, tag it as `dscr.io/username/image:v1` and push it to dscr.io afterwards. 
