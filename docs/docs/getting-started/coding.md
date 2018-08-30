---
title: 3. Coding with your DevSpace
---

Coding within your Devspace is as easy as coding on localhost.

## Terminal Access
Running `devspace up` will open a terminal for you, so you can directly run command within your DevSpace. By default, `devspace up` will use `bash` as shell and fall-back to `sh` if `bash` is not available within your container. 

**Note:** Use `devspace up -s /my/shell` to open the terminal with a different shell.

## Port Forwarding
By default, `devspace up` will forward all TCP and UDP traffic on the ports your application listens on from your localhost machine to the DevSpace within your cluster.

**Note:** See [`/.devspace/config.yaml` configuration](/docs/configuration/config.yaml.html) for details on how to configure more advanced port forwarding procedures.

## Code Synchronization & Hot Reloading
By default, `devspace up` will automatically synchronize your source code to the `/app` folder within your DevSpace. This sync-procedure is bi-directional and allows you to use hot reloading (e.g. for using nodemon for nodejs).

**Note:** See [`/.devspace/config.yaml` configuration](/docs/configuration/config.yaml.html) for details on how to configure more advanced code synchrinization procedures.

## Image Building
When you run `devspace up` for the first time, your `/Dockerfile` will be built automatically. If you run `devspace up` again, it will check if the `/Dockerfile` has been modified since the last build and only re-build if the `/Dockerfile` has changed since then. 

**Note:** To force re-build your docker image, you can run `devspace up -b`.
