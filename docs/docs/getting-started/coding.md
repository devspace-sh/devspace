---
title: 3. Coding with DevSpace
---

There are 5 central features the devspace cli is doing for you when running `devspace up`:
1. Image building 
2. Helm chart deployment 
3. Port forwarding from local machine to remote containers
4. File synchronization between local machine and remote containers
5. Terminal access to your release pods

## Image Building
When you run `devspace up` for the first time, your [/Dockerfile](/docs/configuration/dockerfile.html) will be built automatically. If you run `devspace up` again, it will check if the [/Dockerfile](/docs/configuration/dockerfile.html) has been modified since the last build and only re-build if the [/Dockerfile](/docs/configuration/dockerfile.html) has changed since then. 

**Note:** To force re-build your docker image, you can run `devspace up -b`.

## Chart Deployment
The devspace cli will deploy a helm chart in the target cluster by running `devspace up`. The deployed chart can be found locally in the `chart/` folder and can be changed as wanted. There is one special field in the `chart/values.yaml`: For each container specified under the key `containers.containerName`, the `image` property will be filled automatically after the build step.  

If you are interested how helm charts work and how to write and adjust them, you can take a look at [helm charts](https://github.com/helm/helm/blob/master/docs/charts.md)

## Terminal Access
Running `devspace up` will by default open a terminal for you, so you can directly run commands within your DevSpace. By default, `devspace up` will use `bash` as shell and fall-back to `sh` if `bash` is not available within your container. If you wish to run a different shell or command just execute `devspace up [your command or shell]`  

By running `devspace up` the cli will automatically establish the port forwarding and sync mechanism specified in the [/.devspace/config.yaml](/docs/configuration/config.yaml.html). If you just want to access the release pod, you can also execute `devspace enter` or `devspace enter [command]`.  

## Port Forwarding
By default, `devspace up` will forward all TCP and UDP traffic on the ports your application listens to from your localhost machine to the DevSpace within your cluster. You can see the configured ports by running `devspace list port`. If you want to add a port just run `devspace add port 8080` and on the next `devspace up` the port 8080 will be forwarded from your local machine to the remote port.

**Note:** See [/.devspace/config.yaml](/docs/configuration/config.yaml.html) for details on how to configure more advanced port forwarding procedures.

## Code Synchronization & Hot Reloading
By default a file synchronization path is configured from the project path to the release pod container path `/app`. `devspace up` will then automatically synchronize your source code and remote changes. This allows you to use hot reloading (e.g. for using nodemon for nodejs). You can change this behaviour as you want by configuring the paths in the [/.devspace/config.yaml](/docs/configuration/config.yaml.html)
