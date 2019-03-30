---
title: How to develop with Kubernetes?
---

Running `devspace dev` will do the following:
1. Read the `Dockerfile` and apply in-memory [entrypoint overriding](/docs/cli/development/entrypoint-overrides) (optional)
2. Build a Docker image using the (overridden) `Dockerfile`
3. Push this Docker image to the [DevSpace Container Registry (dscr.io)](/docs/cloud/images/dscr-io)
4. Deploy your Helm chart as defined in `chart/`
5. Start [port forwarding](/docs/cli/development/port-forwarding)
6. Start [real-time code synchronization](/docs/cli/development/synchronization)
7. Start [terminal proxy](/docs/cli/development/terminal)

> It is highly discouraged to run `devspace dev` multiple times in parallel because the port-forwarding as well as the code synchronization processes will interfere with each other.
