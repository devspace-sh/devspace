---
title: Configuring Port Forwarding
sidebar_label: Port Forwarding
id: version-v4.1.0-port-forwarding
original_id: port-forwarding
---

Port-forwarding allows you to access your application on `localhost:[PORT]` by forwarding the network traffic from a localhost port to a specified port of a container.

When starting the development mode, DevSpace starts port-forwarding as configured in the `dev.ports` section of the `devspace.yaml`.
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/devbackend
      - image: john/debugger
dev:
  ports:
  - imageName: backend
    forward:
    - port: 8080
      remotePort: 80
```

Every port-forwarding configuration consists of two parts:
- [Pod/Container Selection](#container-selection)
- [Port Mapping via `port` and optionally via `remotePort`](#port-mapping-devports-forward)

> The `port` option must be unique across your entire `ports` section, e.g. you can only use the value `8080` once for the `port` option in your `ports` section.


## Pod/Container Selection
The following config options are needed to determine the pod to which the traffic should be forwarded:
- [`imageName`](#devports-imagename)
- [`labelSelector`](#devports-labelselector)
- [`namespace`](#devports-namespace)

> If you specify multiple these config options, they will be jointly used to select the pod / container (think logical `AND / &&`).


### `dev.ports[*].imageName`
The `imageName` option expects a string with the name of an image from the `images` section of the `devspace.yaml`. Using `imageName` tells DevSpace to select the container based on the referenced image that was last built using DevSpace.

> Using `imageName` is not possible if multiple deployments use the same image that belongs to this `imageName` referencing the `images` section of the `devspace.yaml`.

#### Example: Select Container by Image Name
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - name: container-0
        image: john/devbackend
      - name: container-1
        image: john/debugger
dev:
  ports:
  - imageName: backend
    forward:
    - port: 8080
      remotePort: 80
  - imageName: backend-debugger
    forward:
    - port: 3000
```
**Explanation:**  
- The above example defines two images that can be used as `imageName`: `backend` and `backend-debugger`
- The deployment starts two containers and each of them uses an image from the `images` section.
- The `imageName` option of the first port-forwarding configuration in the `dev.ports` section references `backend`. That means DevSpace would select the first container for port-forwarding, as this container uses the `image: john/devbackend` which belongs to the `backend` image as defined in the `images` section.
- The `imageName` option of the second port-forwarding configuration in the `dev.ports` section references `backend-debugger`. That means DevSpace would select the second container for port-forwarding, as this container uses the `image: john/debugger` which belongs to the `backend-debugger` image as defined in the `images` section.

In consequence, the following port-forwarding processes would be started when using the above config example:
- `localhost:8080` forwards to `container-0:80`
- `localhost:3000` forwards to `container-1:3000`


### `dev.ports[*].labelSelector`
The `labelSelector` option expects a key-value map of strings with Kubernetes labels.

#### Example: Select Container by Label
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - name: container-0
        image: john/devbackend
      - name: container-1
        image: john/debugger
dev:
  ports:
  - labelSelector:
      app.kubernetes.io/name: devspace-app
      app.kubernetes.io/component: app-backend
      custom-label: custom-label-value
    forward:
    - port: 8080
      remotePort: 80
```
**Explanation:**  
- The `labelSelector` would select the pod created for the component deployment `app-backend`.
- Because containers in the same pod share the same network stack, we do not need to specify which container should be selected.

### `dev.ports[*].namespace`
The `namespace` option expects a string with a Kubernetes namespace used to select the container from.

> It is generally not needed to specify the `namespace` option because by default, DevSpace uses the default namespace of your current kube-context which is usually the one that has been used to deploy your containers to.


## Port Mapping `dev.ports[*].forward`
The `forward` section defines which localhost `port` should be forwarded to the `remotePort` of the selected container.

> By default, `remotePort` will take the same value as `port` if `remotePort` is not explicitly defined.

### `dev.ports[*].forward[*].port`
The `port` option expects an integer from the range of user ports [1024 - 49151].

> Using a `port` < 1024 will likely to cause problems as these ports are reserved as system ports.

> The `port` option is mandatory.

#### Example
**See "[Example: Select Container by Image](#example-select-container-by-image)"**


### `dev.ports[*].forward[*].remotePort`
The `remotePort` option expects an integer from the range of valid ports [0 - 65535].

> By default, `remotePort` has the same value as `port` if `remotePort` is not explictly defined.

#### Example
**See "[Example: Select Container by Image](#example-select-container-by-image)"**

### `dev.ports[*].forward[*].bindAddress`
The `bindAddress` option expects a valid IP address that the local port should be bound to.

#### Default Value For `bindAddress`
```yaml
bindAddress: "0.0.0.0" # listen on all network interfaces
```


<br>

---
## Useful Commands

### `devspace add port`
Use the convenience command `devspace add port [LOCAL_PORT]:[REMOTE_PORT]` to tell DevSpace to forward traffic from your computer's `[LOCAL_PORT]` to the `[REMOTE_PORT]` of the default component of your application. If no `[REMOTE_PORT]` is specified, DevSpace will assume that `[REMOTE_PORT]=[LOCAL_PORT]`. Multiple port mappings can be specified as comma-separated list.
```bash
devspace add port 8080:80,3000
```
The example above would tell DevSpace to forward the local port `8080` to the container port `80` as well as to forward the local port `3000` to the remove container port `3000`.

> Local ports must be unique across all port forwarding configurations.
