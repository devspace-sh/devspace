---
title: /.devspace/config.yaml
---

This is an example of a [.devspace/config.yaml](#)
```yaml
version: v1
portForwarding:
- resourceType: pod
  labelSelector:
    release: my-app
  portMappings:
  - localPort: 3000
    remotePort: 3000
  - localPort: 8080
    remotePort: 80
syncPath:
- resourceType: pod
  labelSelector:
    release: my-app
  localSubPath: ./
  containerPath: /app
registry:
  secrets:
    htpasswd: ""
```
A [.devspace/config.yaml](#) contains any public/shared configuration for running a DevSpace for the respective project. It is highly recommended to put this file under version control (e.g. git add).

## Port Forwarding
To access applications running inside a DevSpace, the DevSpace CLI allows to configure port forwardings. A port forwarding consists of the following:
- `resourceType` (currently only `pod` is supported)
- `labelSelector` (usually the release/app name)
- a list of `portMappings` (each specifying a `localPort` on localhost and a `remotePort` within the DevSpace)

In the example above, you could open `localhost:8080` inside your browser to see the output of the application listening on port 80 within your DevSpace.

## Sync Paths
To comfortably sync code to a DevSpace, the DevSpace CLI allows to configure sync paths. A sync path consists of the following:
- `resourceType` (currently only `pod` is supported)
- `labelSelector` (usually the release/app name)
- `localSubPath` (relative to your local project root)
- `containerPath` (absolute path within your DevSpace)

In the example above, the entire code within the project would be synchronized with the folder `/app` inside the DevSpace.
