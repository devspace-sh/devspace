# Microservices example

This example shows you how to develop two microservices in a single repo with devspace on minikube. The example consists of a small node webserver that makes a http request to a php webserver to retrieve some data.

# Step 0: Prerequisites

1. Install minikube (no docker required, since devspace uses the built in minikube docker daemon)

# Step 1: Start the devspace

1. Run `devspace dev` to start the application in development mode. In development mode the image entrypoint of the node image is overwritten with `sleep 999999999` to avoid the container colliding with the commands you run inside the container (You can change this behaviour in the `.devspace/config.yaml`).

The command does several things in this order:
- Build the docker images (override the entrypoint of node image with sleep 999999999 (don't worry it can still use all cached layers))
- Setup tiller in the namespace
- Deploy the php chart via helm and the node kubernetes manifest via kubectl
- Start port forwarding the remote port 3000 to local port 3000
- Start syncing all files in the node folder with the node container and all files in the php folder with the php container
- Open a terminal to the node pod

```
[info]   Loaded config from .devspace/configs.yaml
[done] √ Create namespace devspace                
[info]   Building image 'node' with engine 'docker'
Sending build context to Docker daemon  5.473kB
Step 1/9 : FROM node:8.11.4
 [...]
[info]   Skip image push for node
[done] √ Done processing image 'node'
[info]   Building image 'php' with engine 'docker'
Sending build context to Docker daemon  8.704kB
Step 1/6 : FROM php:7.1-apache-stretch
[...]
[info]   Skip image push for php
[done] √ Done processing image 'php'
[info]   Deploying devspace-node with kubectl
deployment.extensions/devspace created             
[done] √ Finished deploying devspace-node          
[info]   Deploying devspace-php with helm
[done] √ Created deployment tiller-deploy in devspace
[done] √ Tiller started                     
[done] √ Deployed helm chart (Release revision: 1)                    
[done] √ Finished deploying devspace-php
[done] √ Port forwarding started on 3000:3000           
[done] √ Sync started on /covexo/devspace/examples/microservices/node <-> /app (Pod: devspace/devspace-798ff95944-dn2jj)
[done] √ Sync started on /covexo/devspace/examples/microservices/php <-> /var/www/html (Pod: devspace/devspace-php-5c7d99565c-l5f62)
root@devspace-798ff95944-dn2jj:/app#
```
2. Run `npm start` in the new opened terminal to start the webserver
3. Go to `localhost:3000` to see the output of the webserver
4. Change the message in the `php/index.php`
5. Refresh the browser to see the changes applied.

# Troubleshooting 

If you experience problems during deploy or want to check if there are any issues within your deployed application devspace provides useful commands for you:
- `devspace analyze` analyzes the namespace and checks for warning events and failed pods / containers
- `devspace enter` open a terminal to a kubernetes pod (the same as running `kubectl exec ...`)
- `devspace logs` shows the logs of a devspace (the same as running `kubectl logs ...`)
- `devspace purge` delete the deployed application

See https://devspace.cloud/docs for more advanced documentation
