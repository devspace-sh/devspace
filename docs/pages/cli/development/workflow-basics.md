---
title: Workflow & Basics
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
1. Read your application's Dockerfiles and apply in-memory [entrypoint overrides](/docs/development/overrides#configuring-entrypoint-overrides) (optional)
2. Build your application's Dockerfiles as specified in your `devspace.yaml`
3. Push the resulting Docker images to the registries specified in your `devspace.yaml`
4. Deploy your application similar to using `devspace deploy`
5. Start [port forwarding](/docs/development/port-forwarding)
6. Start [real-time code synchronization](/docs/development/synchronization)
7. Start [terminal proxy](/docs/development/terminal) (optional, [see how to configure log streaming instead](/docs/development/terminal#print-logs-instead-of-opening-a-terminal))

<img src="/img/processes/development-process-devspace.svg" alt="DevSpace Development Process" style="width: 100%;">

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
- [dev.overrideImages](/docs/development/overrides)
- [dev.ports](/docs/development/port-forwarding)
- [dev.sync](/docs/development/synchronization)
- [dev.terminal](/docs/development/terminal)











DevSpace CLI provides useful features for developing your application directly in kubernetes. The command `devspace dev` can be used to start an application in development mode and starts services like file synchronization, port forwarding and terminal proxying automatically based on the configuration.  

The development experience is very similar to using `docker-compose`, so if you are already familiar on how to develop with `docker-compose`, DevSpace will behave very similar. One of the major benefits of DevSpace versus docker-compose is that DevSpace allows you to develop in any kubernetes cluster, either locally or in any remote kubernetes cluster.   
  
When running `devspace dev` will do the following:
1. Read the `Dockerfile` and apply in-memory [entrypoint overriding](/docs/development/overrides) (optional)
2. Build and push Docker images using the (overridden) `Dockerfile`
3. Deploy the [deployments](/docs/workflow-basics/deployment) defined in `devspace.yaml`
4. Start [port forwarding](/docs/development/port-forwarding)
5. Start [real-time code synchronization](/docs/development/synchronization)
6. Start a single [terminal proxy](/docs/development/terminal)

> It is highly discouraged to run `devspace dev` multiple times in parallel because multiple instances of port-forwarding and code synchronization will disturb each other. Run `devspace enter` to open additional terminals without port-forwarding and code synchronization.

If you are using **DevSpace in a team**, DevSpace also allows you to define [variables](/docs/configuration/variables) in your configuration that are filled dynamically during development based on user input, environment variables or other runtime specific circumstances. This can be very helpful to build a common config that can be shared accross your team and checked into a version control system, but still behaves differently for each developer.  

If you want to allow your developers to develop applications inside a single cluster, you should also take a look at [DevSpace Cloud Spaces](/docs/cloud/spaces/what-are-spaces). They are essentially flexible isolated kubernetes namespaces that can be spinned up and shutdown by the user itself.



TODO: ENV VARS
