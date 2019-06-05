---
title: How to develop with Kubernetes?
---

DevSpace CLI provides useful features for developing your application directly in kubernetes. The command `devspace dev` can be used to start an application in development mode and starts services like file synchronization, port forwarding and terminal proxying automatically based on the configuration.  

The development experience is very similar to using `docker-compose`, so if you are already familiar on how to develop with `docker-compose`, DevSpace will behave very similar. One of the major benefits of DevSpace versus docker-compose is that DevSpace allows you to develop in any kubernetes cluster, either locally or in any remote kubernetes cluster.   
  
When running `devspace dev` will do the following:
1. Read the `Dockerfile` and apply in-memory [entrypoint overriding](/docs/development/entrypoint-overrides) (optional)
2. Build and push Docker images using the (overridden) `Dockerfile`
3. Deploy the [deployments](/docs/workflow-basics/deployment) defined in `devspace.yaml`
4. Start [port forwarding](/docs/development/port-forwarding)
5. Start [real-time code synchronization](/docs/development/synchronization)
6. Start a single [terminal proxy](/docs/development/terminal)

> It is highly discouraged to run `devspace dev` multiple times in parallel because multiple instances of port-forwarding and code synchronization will disturb each other. Run `devspace enter` to open additional terminals without port-forwarding and code synchronization.

If you are using **DevSpace in a team**, DevSpace also allows you to define [variables](/docs/configuration/variables) in your configuration that are filled dynamically during development based on user input, environment variables or other runtime specific circumstances. This can be very helpful to build a common config that can be shared accross your team and checked into a version control system, but still behaves differently for each developer.  

If you want to allow your developers to develop applications inside a single cluster, you should also take a look at [DevSpace Cloud Spaces](/docs/cloud/spaces/what-are-spaces). They are essentially flexible isolated kubernetes namespaces that can be spinned up and shutdown by the user itself.
