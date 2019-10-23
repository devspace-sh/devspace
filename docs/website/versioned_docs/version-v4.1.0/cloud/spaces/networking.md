---
title: Networking (Domains)
id: version-v4.1.0-networking
original_id: networking
---

In a [Space](../../cloud/spaces/what-are-spaces), applications can be accessed through [port-forwarding](../../cli/development/configuration/port-forwarding) on localhost or with [ingresses](../../cli/deployment/workflow-basics) on a certain domain.

The easiest way to create an ingress to route traffic from the internet to your containers inside Kubernetes is to run:
```bash
devspace open
```
