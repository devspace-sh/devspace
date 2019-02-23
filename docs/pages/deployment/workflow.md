---
title: Workflow & basics
---

Running `devspace deploy` will do the following:
1. Build all Docker [images that you specified in `images` within `.devspace/config.yaml`](../deployment/images)
2. Push the Docker images to the [DevSpace Container Registry (dscr.io)](../images/internal-registry) or to any [external registry](../images/external-registries)
3. Deploy all [deployments defined in `.devspace/config.yaml`](../deployment/deployments)
