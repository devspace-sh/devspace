---
title: Resource limits
id: version-v3.5.18-resource-limits
original_id: resource-limits
---

Components allow you to use Kubernetes capabilities to allocate and limit computing resources for containers. Generally, there are two types of resource settings:
- **Resource limits** define the maximum amount of resources a container can use
- **Resource requests** define an amount of resources that will be allocated/reserved for a container which cannot be used by any other container

Resource limits need to be equal or higher than resource requests.

> DevSpace Cloud automatically adds default resource limits to your containers to ensure that a user does not accentially deploy containers with use up a lot of the available resources in a cluster, leaving other users with the problem that their containers cannot be started anymore. You can adjust theses standard limits in the UI.

## Types of resources
You can set resource limits and resource requests for the following resources:
- CPU in Core units, i.e. 3 = 3 Cores, 800m = 0.8 Core (=800 **M**illi-Core)
- Memory (RAM) in Gi (Gigabyte), Mi (Megabyte) or Ki (Kilobyte)
- [Emphemeral (non-persistent container) storage (?)](#what-is-ephemeral-storage) in Gi (Gigabyte), Mi (Megabyte) or Ki (Kilobyte)

## Define resource limits
To limit the resources of a container, simply configure the `limits` within the `resources` section of the container.
```yaml
deployments:
- name: backend
  component:
    containers:
    - image: dscr.io/username/api-server
      resources:
        limits:
          cpu: 400m
          memory: 500Mi
          ephemeralStorage: 2Gi
```
The above example would define that this container can use a maximum of:
- 0.4 Cores
- 500 Megabytes of Memory (RAM)
- 2 Gigabytes of [ephemeral storage (?)](#what-is-ephemeral-storage)

> Resource limits should always be higher than the resource requests. Because resource limits are not allocated/reserved for a container (unlike resource requests), it is possible to oversubscribe resource limits, i.e. use a total of resource limits over all containers which is more than the cluster has.

## Define resource requests
To allocate/reserve resources for a container, simply configure the `requests` within the `resources` section of the container.
```yaml
deployments:
- name: backend
  component:
    containers:
    - image: dscr.io/username/api-server
      resources:
        requests:
          cpu: 200m
          memory: 300Mi
          ephemeralStorage: 1Gi
```
The above example would define that this container can use a maximum of:
- 0.2 Cores
- 300 Megabytes of Memory (RAM)
- 1 Gigabytes of [ephemeral storage (?)](#what-is-ephemeral-storage)


## FAQ

<details>
<summary>
### What is ephemeral storage?
</summary>
Ephemeral storage is the non-persistent storage of a container, i.e. the storage used within the root partition `/` of a container. 

If you save a file in a [(persistent) volume](/docs/deployment/components/configuration/volumes), it will not add to the epemeral storage but if you add it to a folder which does not belong to a volume, it will be count as ephemeral storage.
</details>
