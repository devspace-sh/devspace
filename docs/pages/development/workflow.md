---
title: Workflow & basics
---

Running `devspace dev` will do the following:
1. Read the `Dockerfile` and apply in-memory [entrypoint overriding](../development/entrypoint-overrides) (optional)
2. Build a Docker image using the (overridden) `Dockerfile`
3. Push this Docker image to the [DevSpace Container Registry (dscr.io)](../images/internal-registry)
4. Deploy your Helm chart as defined in `chart/`
5. Start [port forwarding](../development/port-forwarding)
6. Start [real-time code synchronization](../development/synchronization)
7. Start [terminal proxy](../development/terminal)

> It is highly discouraged to run `devspace dev` multiple times in parallel because the port-forwarding as well as the code synchronization processes will interfere with each other.
