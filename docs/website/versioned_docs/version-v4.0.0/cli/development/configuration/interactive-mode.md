---
title: Configuring Interactive Mode
sidebar_label: Interactive Mode
id: version-v4.0.0-interactive-mode
original_id: interactive-mode
---

The development mode of DevSpace can be started using the `-i / --interactive` flag which overrides the `ENTRYPOINT` of an image with `[sleep, 999999]` and opens an interactive terminal session for one of the containers that use the 'sleeping' image. Due to the `ENTRYPOINT` override, the application has not been started within the container and the user can start the application manually through the interactive terminal session.

To control the default behavior when using the interactive mode, you can configure the `dev.interactive` section in the `devspace.yaml`.
```yaml
images:
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
  backend:
    image: john/appbackend
  database:
    image: john/database
deployments:
- name: app-frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
- name: app-backend
  component:
    containers:
    - image: john/appbackend
      name: some-container
    - image: john/appbackend
dev:
  interactive:
    images:
    - name: backend
      entrypoint:
      - tail
      cmd:
      - -f
      - /dev/null
    - name: frontend
      entrypoint:
      - /debug_entrypoing.sh
    terminal:
      imageName: backend
      containerName: some-container
```

## `dev.interactive.images`
The `images` option expects an array of objects having the following properties:
- `name` stating an image name which references an image in the `images` within `devspace.yaml` (required)
- `entrypoint` defining an [`ENTRYPOINT` override](http://localhost:3000/docs/cli/image-building/configuration/overview-specification#images-entrypoint) that will be applied only in interactive mode (optional)
- `cmd` defining a [`CMD` override](http://localhost:3000/docs/cli/image-building/configuration/overview-specification#images-cmd) that will be applied only in interactive mode (optional)

> `ENTRYPOINT` and `CMD` overrides for interactive mode work the same way as regular [overrides for image building](/docs/cli/image-building/configuration/overview-specification#overriding-entrypoint-cmd). However, they take precedence over regular overrides and are only used during interactive mode.

> By default, DevSpace asks which image to override when starting interactive mode (or skips the question if only one image is defined). This image will then be built using the `ENTRYPOINT [sleep, 999999]` override if not configured differently via `dev.interactive.images`.

#### Example: Setting Interactive Mode Images
```yaml
images:
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
  backend:
    image: john/appbackend
  database:
    image: john/database
deployments:
- name: app-frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
- name: app-backend
  component:
    containers:
    - image: john/appbackend
      name: some-container
    - image: john/appbackend-sidecar
dev:
  interactive:
    images:
    - name: backend
      entrypoint:
      - tail
      cmd:
      - -f
      - /dev/null
    - name: frontend
      entrypoint:
      - /debug_entrypoing.sh
```
**Explanation:**  
- The above example defines 3 images and 2 deployments.
- The `dev.interactive.images` option defines that the image `backend` should be built using a `ENTRYPOINT [tail]` and using `CMD [-f, /dev/null]` when building this image in interactive mode
- The `dev.interactive.images` option defines that the image `frontend` should be built using a `ENTRYPOINT [7debug_entrypoint.sh]` when building this image in interactive mode


## `dev.interactive.terminal`
The `terminal` option expects an objects having the following container selection properties:
- `imageName` to select a container based on an image specified in `images` (cannot be used in combiation with `labelSelector`)
- `labelSelector` to select a pod based a Kubernetes label selector (cannot be used in combiation with `imageName`)
- `containerName` to select a container based on its name (optional when `labelSelector` is used)
- `namespace` to select a container from a namespace different than the default namespace of the current kube-context

> You can set **either** `labelSelector` (optionally in combiantion with `containerName`) **or** `imageName`. Both options can be combined with the optional `namespace` option if needed.

Additionally, the object expected in `terminal` has the following non-selection property:
- `command` defines a command to run when starting the terminal session (default: `/bin/bash` with fallback `/bin/sh`)

> If `command` is a non-interactive command that terminates, DevSpace will run the command and exit after the command has terminated.

#### Example: Configuring Terminal for Interactive Mode
```yaml
images:
  frontend:
    image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
  backend:
    image: john/appbackend
  database:
    image: john/database
deployments:
- name: app-frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
- name: app-backend
  component:
    containers:
    - image: john/appbackend
      name: some-container
    - image: john/appbackend-sidecar
dev:
  interactive:
    terminal:
      imageName: backend
      containerName: some-container
```
**Explanation:**  
The above configuration would open the container with name `some-container` that belongs to the deployment `app-backend` when running `devspace dev -i`.


## `dev.interactive.defaultEnabled`
The `defaultEnabled` option expects a boolean that determines if interactive mode should be started by default even if no `-i / --interactive` flag was provided.

#### Default Value For `defaultEnabled`
```yaml
defaultEnabled: false
```

#### Example: Enabling Interactive Mode By Default
```yaml
images:
  backend:
    image: john/appbackend
deployments:
- name: app-backend
  component:
    containers:
    - image: john/appbackend
      name: some-container
    - image: john/appbackend-sidecar
dev:
  interactive:
    defaultEnabled: true
    images:
    - name: backend
      entrypoint:
      - /debug_entrypoing.sh
    terminal:
      imageName: backend
      containerName: some-container
```
**Explanation:**  
Running `devspace dev` with the above configuration leads to:
- during image building: the overrides defined in `dev.interactive.images`  would be applied
- after deployment: an interactive terminal session for a container with name `some-container` and with image `ohn/appbackend` would be started
