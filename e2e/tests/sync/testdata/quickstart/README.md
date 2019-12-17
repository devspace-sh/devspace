# Quickstart example

This example shows you how to develop and deploy a small node express application with devspace on devspace.cloud. For a more detailed documentation, take a look at https://devspace.cloud/docs

# Step 0: Prerequisites

In order to get things ready do the following:
1. Install docker
2. Run `devspace create space quickstart` to create a new kubernetes namespace in the devspace.cloud (if you want to use your own cluster just skip this step and devspace will use the current kubectl context, you probably also want to exchange the image name in `devspace.yaml`)

# Step 1: Develop the application

1. Run `devspace dev` to start the application in development mode. In development mode the image entrypoint is overwritten with `sleep 999999999` to avoid the container colliding with the commands you run inside the container (You can change this behaviour in the `devspace.yaml`).

The command does several things in this order:
- Build the docker image (override the entrypoint with sleep 999999 (don't worry it can still use all cached layers))
- Setup tiller in the namespace and create needed image pull secrets
- Deploy the component (an easy to use helm chart), you can also deploy your custom helm chart, kustomize or regular kubectl manifests
- Start port forwarding the remote port 3000 to local port 3000
- Start syncing all files in the quickstart folder with the remote container
- Open a terminal to the remote pod

You should see the following output:
```
[info]   Loaded config from devspace.yaml
[info]   Using space quickstart                       
[info]   Building image 'dscr.io/yourname/devspace' with engine 'docker'
[done] √ Authentication successful (dscr.io)
Sending build context to Docker daemon  9.031kB
Step 1/9 : FROM node:8.11.4
 ---> 8198006b2b57
[...]
[info]   Image pushed to registry (dscr.io)
[done] √ Done processing image 'dscr.io/yourname/devspace'
[info]   Deploying devspace-app with helm
[done] √ Deployed helm chart (Release revision: 4)                                            
[done] √ Finished deploying devspace-app
[done] √ Port forwarding started on 3000:3000           
[done] √ Sync started on /devspace-cloud/devspace/examples/quickstart <-> /app (Pod: d4c1654922db400f612a027283b50001/default-7c4dcdfc4-m867d)
[info]   The Space is now reachable via ingress on this URL: https://yourname.devspace.host
root@default-7c4dcdfc4-m867d:/app#
```
2. Run `npm start` in the new opened terminal to start the webserver
3. Go to `localhost:3000` to see the output of the webserver (or https://yourname.devspace.host)
4. Change something in the `index.js`
5. You should see the webserver restarting
6. Refresh the browser to see the changes applied.

# Step 2: Deploy the application

Deploying the application is the same as developing it, but instead of `devspace dev` you run `devspace deploy`. The deploy command does not override the image entrypoint and does not start any of the developing services (port-forwarding, sync and terminal) and just deploys the application.

# Troubleshooting 

If you experience problems during deploy or want to check if there are any issues within your deployed application devspace provides useful commands for you:
- `devspace analyze` analyzes the namespace and checks for warning events and failed pods / containers
- `devspace enter` open a terminal to a kubernetes pod (the same as running `kubectl exec ...`)
- `devspace logs` shows the logs of a devspace (the same as running `kubectl logs ...`)
- `devspace purge` delete the deployed application
- `devspace open` to create an ingress and open the application in the browser

See https://devspace.cloud/docs for more advanced documentation
