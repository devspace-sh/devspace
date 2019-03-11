---
title: Workflow & basics
---

Running `devspace deploy` will do the following:
1. Build all Docker [images that you specified in `images` within `.devspace/config.yaml`](/docs/cli/deployment/images)
2. Push the Docker images to any [Docker registry](/docs/cli/images/workflow)
3. Deploy all [deployments defined in `.devspace/config.yaml`](/docs/cli/deployment/deployments)
