---
title: Configure Component Deployments
sidebar_label: Components
---

To deploy a component, you need to configure it within the `deployments` section of the `devspace.yaml`.
```yaml
#TODO
```

[What are components?](#TODO)


## Containers & Pods

### `containers`
The `image` option expects a string containing the image repository including registry and image name. 

- Make sure you [authenticate with the image registry](/docs/cli/image-building/workflow-basics#registry-authentication) before using in here.
- For Docker Hub images, do not specify a registry hostname and use just the image name instead (e.g. `mysql`, `my-docker-username/image`).

#### Example: Multiple Images
```yaml
images:
  backend:
    image: john/appbackend
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
```
**Explanation:**
- The first image `backend` would be tagged as `appbackend:[TAG]` pushed to Docker Hub using the path `john` (which generally could be your Docker Hub username).
- The second image `frontend` would be tagged as `appfrontend:[TAG]` and pushed to `dscr.io` using the path `${DEVSPACE_USERNAME}` which is a [dynamic config variable](#TODO) that resolves to your username in DevSpace Cloud. 

> See **[`images[*].tag` *Tagging Schema*](#images-tag-tagging-schema)** for details on how the image `[TAG]` would be set in this case.


### `labels`
### `annotations`

## Volumes & Persistent Storage
### `volumes`

## Service & In-Cluster Networking
### `service`
### `serviceName`

## Ingress & Domain
### `ingress`

## Scaling
### `replicas`
### `autoScaling`

## Advanced
### `rollingUpdate`
### `pullSecrets`
### `podManagementPolicy`
### `options`
