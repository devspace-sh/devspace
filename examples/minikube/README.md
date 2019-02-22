# Minikube example

This example shows you how to develop a small node express application with devspace on minikube. For a more detailed documentation, take a look at https://devspace.cloud/docs

# Step 0: Prerequisites

1. Install minikube (no docker required, since devspace uses the built in minikube docker daemon)

# Step 1: Develop the application

1. Run `devspace dev` to start the application in development mode. In development mode the image entrypoint is overwritten with `sleep 999999999` to avoid the container colliding with the commands you run inside the container (You can change this behaviour in the `.devspace/config.yaml`).

The command does several things in this order:
- Build the docker image (override the entrypoint with sleep 999999999 (don't worry it can still use all cached layers))
- Setup tiller in the namespace and create needed image pull secrets
- Deploy the chart (you can find and change it in `chart/`)
- Start port forwarding the remote port 3000 to local port 3000
- Start syncing all files in the minikube folder with the remote container
- Open a terminal to the remote pod

You should see the following output:
```
[info]   Loaded config from .devspace/configs.yaml
[done] √ Create namespace devspace                
[info]   Building image 'devspace' with engine 'docker'
Sending build context to Docker daemon  7.498kB
Step 1/11 : FROM node:8.11.4
[...]
[info]   Skip image push for devspace
[done] √ Done processing image 'devspace'
[info]   Deploying devspace-app with helm
[done] √ Created deployment tiller-deploy in devspace
[done] √ Tiller started                     
[done] √ Deployed helm chart (Release revision: 1)                    
[done] √ Finished deploying devspace-app
[done] √ Port forwarding started on 3000:3000           
[done] √ Sync started on /covexo/devspace/examples/minikube <-> /app (Pod: devspace/default-f5c8cbcd6-w29rs)
root@default-f5c8cbcd6-w29rs:/app#
```
2. Run `npm start` in the new opened terminal to start the webserver
3. Go to `localhost:3000` to see the output of the webserver
4. Change something in the `index.js`
5. You should see the webserver restarting
6. Refresh the browser to see the changes applied.

# Troubleshooting 

If you experience problems during deploy or want to check if there are any issues within your deployed application devspace provides useful commands for you:
- `devspace analyze` analyzes the namespace and checks for warning events and failed pods / containers
- `devspace enter` open a terminal to a kubernetes pod (the same as running `kubectl exec ...`)
- `devspace logs` shows the logs of a devspace (the same as running `kubectl logs ...`)
- `devspace purge` delete the deployed application

See https://devspace.cloud/docs for more advanced documentation
