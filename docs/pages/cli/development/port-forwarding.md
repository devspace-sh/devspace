---
title: Configure port forwarding
---

When running `devspace dev`, DevSpace CLI will start forwarding traffic of the ports on your local computer and the ports within the containers inside your Space. This allows you to access containers via `localhost` instead of having to expose them via `services` and `ingresses`.

## Add a port forwarding configuration
Use the convenience command `devspace add port [LOCAL_PORT]:[REMOTE_PORT]` to tell DevSpace CLI to forward traffic from your computer's `[LOCAL_PORT]` to the `[REMOTE_PORT]` of the default component of your application. If no `[REMOTE_PORT]` is specified, DevSpace CLI will assume that `[REMOTE_PORT]=[LOCAL_PORT]`. Multiple port mappings can be specified as comma-separated list.
```bash
devspace add port 8080:80,3000
```
The example above would tell DevSpace CLI to forward the local port `8080` to the container port `80` as well as to forward the local port `3000` to the remove container port `3000`.

> Local ports must be unique across all port forwarding configurations.

## Configure port forwarding
The configuration for port forwarding can be set within the `dev.ports` section of `.devspace/config.yaml`.
```yaml
dev:
  ports:
    selector: default
    portMappings:
    - localPort: 8080
      remotePort: 80
    - localPort: 3000
      remotePort: 3000
  selectors:
  - name: default
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
```
The above example shows the port forwarding configuration that would be created when running the exemplary `devspace add port` command as shown above.

## Remove a port forwarding configuration
Use the convenience command `devspace remove port [LOCAL_PORT]:[REMOTE_PORT]` to remove a port forwarding configuration.
```bash
devspace remove port 8080:80,3000
```
This exemplary command would remove the port forwarding configurations created by the `devspace add port` command shown above.
