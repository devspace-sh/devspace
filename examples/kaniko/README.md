# Kaniko example

This example shows how kaniko can be used instead of docker to build and push an docker image directly inside the cluster.   

# Step 0: Prerequisites

1. Install minikube
2. Exchange the image `yourdockeruser/devspace` in `devspace.yaml` and `chart/values.yaml` with your docker username 

# Step 1: Start the devspace

1. Run `devspace dev` to start the application in development mode. In development mode the image entrypoint is overwritten with `sleep 999999999` to avoid the container colliding with the commands you run inside the container (You can change this behaviour in the `devspace.yaml`).

The command does several things in this order:
- Build the docker image via kaniko build pod and override the entrypoint with sleep 999999999 
- Setup tiller in the namespace and create needed image pull secrets
- Deploy the chart (you can find and change it in `chart/`)
- Start port forwarding the remote port 8080 to local port 8080
- Start syncing all files in the kaniko folder with the remote container
- Open a terminal to the remote pod

You should see the following output:
```
[info]   Loaded config from devspace.yaml
[info]   Building image 'devspacecloud/devspace' with engine 'kaniko'
[done] √ Authentication successful (hub.docker.com)
[done] √ Kaniko build pod started                        
[done] √ Uploaded files to container 
[info]   build >>> INFO[0002] Downloading base image golang:1.11           
[...]
[info]   build >>> 2019/02/19 04:28:31 index.docker.io/devspacecloud/devspace:31Cdw93: digest: sha256:5a85bf49845b9bb9ef70819e58a641b45266cec07e4def0947c224d00137700b size: 1578
[done] √ Done building image                
[info]   Image pushed to registry (hub.docker.com)
[done] √ Done processing image 'devspacecloud/devspace'
[info]   Deploying devspace-default with helm
[done] √ Deployed helm chart (Release revision: 2)                    
[done] √ Finished deploying devspace-default
[done] √ Port forwarding started on 8080:8080           
[done] √ Sync started on /devspace-cloud/devspace/examples/kaniko <-> /app (Pod: devspace/default-7bf98f5d86-xpqmr)
root@default-7bf98f5d86-xpqmr:/go/src/app#
```
2. Run `go run main.go` in the new opened terminal to start the webserver
3. Go to `localhost:8080` to see the output of the webserver
4. Change the message in `main.go`
5. Restart the server with `go run main.go`
6. Refresh the browser to see the changes applied.

# Troubleshooting 

If you experience problems during deploy or want to check if there are any issues within your deployed application devspace provides useful commands for you:
- `devspace analyze` analyzes the namespace and checks for warning events and failed pods / containers
- `devspace enter` open a terminal to a kubernetes pod (the same as running `kubectl exec ...`)
- `devspace logs` shows the logs of a devspace (the same as running `kubectl logs ...`)
- `devspace purge` delete the deployed application

See https://devspace.cloud/docs for more advanced documentation
