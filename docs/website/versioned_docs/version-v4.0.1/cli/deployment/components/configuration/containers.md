---
title: Containers
id: version-v4.0.1-containers
original_id: containers
---

## Container Images
Components deploy pods which are a set of containers. These containers are created based on Docker images. To define the image for a container, simply set the `image` value for the container:
```yaml
deployments:
- name: my-backend
  component:
    containers:
    - image: dscr.io/username/my-backend-image
    - image: nginx:1.15
```
The example above would create a pod with two containers:
1. The first container would be create from the image `dscr.io/username/my-backend-image`
2. The second container would be created from the `nginx` image on [Docker Hub](https://hub.docker.com) which is tagged as version `1.15`

> If you are using a private Docker registry, make sure to [logged into this registry](../../../../image-building/registries/authentication).


## Entrypoint (cmd, args)
Components allow you to use the Kubernetes feature of overriding the container startup commands:
- `command` which will override the `ENTRYPOINT` specified in the Dockerfile
- `args` which will override the `CMD` specified in the Dockerfile

```yaml
deployments:
- name: backend
  component:
    containers:
    - image: dscr.io/username/api-server
      command:
      - sleep
      args:
      - 9999999
```
The above example would start the container effectively with the following command: `sleep 9999999`

For more information, please take a look at the [Kubernetes documentation for setting command and args](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/).


## Environment Variables
Instead of storing configuration data (e.g. database host, username and password) inside your Docker image, you should define such information as environment variables.

You can define environment variables for your containers in the `env` section the each container within `devspace.yaml`.
```yaml
deployments:
- name: database
  component:
    containers:
    - image: "dscr.io/username/mysql"
      env:
      - name: MYSQL_USER
        value: "my_user"
      - name: MYSQL_PASSWORD
        value: "my-secret-passwd"
```
The above example would set two environment variables, `MYSQL_USER="my_user"` and `MYSQL_PASSWORD="my-secret-passwd"` within the first container of the `database` component.

<details>
<summary>
### View the specification for environment variables
</summary>
```yaml
name: [a-z0-9-]{1,253}      # Name of the environment variable
value: [string]             # Option 1: Set static value for the environment variable
valueFrom:                  # Option 2: Load value from another resource
  secretKeyRef:             # Option 2.1: Use the content of a Kubernetes secret as value
    name: [secret-name]     # Name of the secret
    key: [key-name]         # Key within the secret
  configMapKeyRef:          # Option 2.2: Use the content of a Kubernetes configMap as value
    name: [configmap-name]  # Name of the config map
    key: [key-name]         # Key within the config map
```

The value of an environment variable can be either set:
1. By directly inserting the value via `value`
2. By referencing a key within a secret via `valueFrom.secretKeyRef`
3. By referencing a key within a configMap via `valueFrom.configMapKeyRef`
4. By using any other field supported for `valueFrom` as defined by the [Kubernetes specification for `v1.EnvVarSource`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#envvarsource-v1-core)
</details>

## Volume Mounts
See [Volumes](../../../../cli/deployment/components/configuration/volumes#mount-volumes-into-containers) for details.


## Resource Limits &amp; Requests
Components allow you to use Kubernetes capabilities to allocate and limit computing resources for containers. Generally, there are two types of resource settings:
- **Resource limits** define the maximum amount of resources a container can use
- **Resource requests** define an amount of resources that will be allocated/reserved for a container which cannot be used by any other container

Resource limits need to be equal or higher than resource requests.

> DevSpace Cloud automatically adds default resource limits to your containers to ensure that a user does not accentially deploy containers with use up a lot of the available resources in a cluster, leaving other users with the problem that their containers cannot be started anymore. You can adjust theses standard limits in the UI.

### Types of Resources
You can set resource limits and resource requests for the following resources:
- CPU in Core units, i.e. 3 = 3 Cores, 800m = 0.8 Core (=800 **M**illi-Core)
- Memory (RAM) in Gi (Gigabyte), Mi (Megabyte) or Ki (Kilobyte)
- [Emphemeral (non-persistent container) storage (?)](#what-is-ephemeral-storage) in Gi (Gigabyte), Mi (Megabyte) or Ki (Kilobyte)

### Set Resource Limits
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

If you save a file in a [(persistent) volume](../../../../deployment/components/configuration/volumes), it will not add to the epemeral storage but if you add it to a folder which does not belong to a volume, it will be count as ephemeral storage.
</details>


## Health checks
Components allow you to use the Kubernetes feature of defining health checks:
- `livenessProbe` allows Kubernetes to check wheather the container is running correctly and restart/recreate it if necessary
- `readinessProbe` allows Kubernetes to check when the container is ready to accept requests (e.g. becoming ready after completing initial startup tasks)

```yaml
deployments:
- name: backend
  component:
    containers:
    - image: dscr.io/username/api-server
      livenessProbe:
        httpGet:
          path: /healthz
          port: 8080
          httpHeaders:
          - name: Custom-Header
            value: Awesome
        initialDelaySeconds: 3
        periodSeconds: 3
      readinessProbe:
        exec:
          command:
          - cat
          - /tmp/healthy
        initialDelaySeconds: 5
        periodSeconds: 5
```
The above example would define an [HTTP livenessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-a-liveness-http-request) and an [exec readinessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-readiness-probes) for the container. Components allow you to use all capabilities for livenessProbes and readinessProbes that the Kubernetes specification provides.

For more information, please take a look at the [Kubernetes documentation for configuring liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/).
