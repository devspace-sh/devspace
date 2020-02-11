---
title: Develop with DevSpace
id: version-v3.5.18-development
original_id: development
---

DevSpace CLI lets you build applications directly inside a Kubernetes cluster with this command:
```bash
devspace dev
```
The configuration for this command can be found in the `dev` section within your `devspace.yaml`.

> It is highly discouraged to run `devspace dev` multiple times in parallel because multiple instances of port-forwarding and code synchronization will disturb each other. Run `devspace enter` to open additional terminals without port-forwarding and code synchronization.

## Use cases for Kubernetes-based development
Kubernetes-based development can be useful in the following cases:
- Your applications needs to access cluster-internal services (e.g. Cluster DNS)
- You want to test your application in a production-like environment
- You want to debug issues that are hard to reproduce on your local machine

The biggest advantages of developing directly inside Kubernetes is that your dev environment will be very similar to your production environment and you can have a much greater confidence that everything will work in production when shipping new features.

## Development process
Running `devspace dev` will do the following:
1. Read your application's Dockerfiles and apply in-memory [entrypoint overrides](../development/overrides#configuring-entrypoint-overrides) (optional)
2. Build your application's Dockerfiles as specified in your `devspace.yaml`
3. Push the resulting Docker images to the registries specified in your `devspace.yaml`
4. Deploy your application similar to using `devspace deploy`
5. Start [port forwarding](../development/port-forwarding)
6. Start [real-time code synchronization](../development/synchronization)
7. Start [terminal proxy](../development/terminal) (optional, [see how to configure log streaming instead](../development/terminal#print-logs-instead-of-opening-a-terminal))

## Useful commands
| Command&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; | Important flags |
|---|---|
|`devspace dev`<br> <small>Starts the development mode</small> | `-b • Rebuild images (force)` <br> `-d • Redeploy everything (force)` |
|`devspace enter`<br> <small>Opens a terminal session for a container</small> | `-p • Pick a container instead of using the default one` |
|`devspace enter [command]`<br> <small>Runs a command inside a container</small> | |
|`devspace logs` <br> <small>Prints the logs of a container</small> | `-p • Pick a container instead of using the default one` <br> `-f • Stream new logs (follow/attach)` |
|`devspace analyze` <br> <small>Analyzes your deployments for issues</small> |  |

## Configuration options
DevSpace CLI lets you define the following types of deployments:
- [dev.overrideImages](../development/overrides)
- [dev.ports](../development/port-forwarding)
- [dev.sync](../development/synchronization)
- [dev.terminal](../development/terminal)
