# Redeploy Instead of Hot Reload example

This example shows you how to develop a small go application with automated repeated deploying with devspace on devspace.cloud. For a more detailed documentation, take a look at https://devspace.cloud/docs

# Step 0: Prerequisites

In order to get things ready do the following:
1. Install docker
2. Install kubectl (https://kubernetes.io/docs/tasks/tools/install-kubectl/)
3. Run `devspace login`
4. Exchange the image `dscr.io/yourusername/devspace` in `.devspace/config` and `kube/deployment.yaml` with your username (you can also use a different registry, but make sure you are logged in with `docker login`)
5. Run `devspace create space quickstart` to create a new kubernetes namespace in the devspace.cloud (if you want to use your own cluster just erase the cloudProvider in `.devspace/config` and skip this step)

# Step 1: Develop the application

1. Run `devspace dev` to start the application. The output should be similar to this

The command does several things in this order:
- Build the docker image
- Create needed image pull secrets
- Deploy the kubectl manifests (you can find and change them in `kube/`)
- Attach to the pod and print its output

You should see the following output:
```
[info]   Loaded config from .devspace/configs.yaml
[info]   Using space fabian                       
[info]   Skip building image 'default'         
[info]   Deploying devspace-default with kubectl
deployment.extensions/devspace unchanged           
[done] √ Finished deploying devspace-default       
[info]   The Space is now reachable via ingress on this URL: https://fabian.devspace.host
[info]   Will now try to print the logs of a running devspace pod...
[info]   Printing logs of pod devspace-59c4d868f8-j2pg5/default...
Hello World!
Hello World!
Hello World!
Hello World!
```
2. Change the message in main.go and you should see the container reloading 
```
[info]   Change detected, will reload in 2 seconds
[info]   Building image 'dscr.io/fabiankramm/devspace' with engine 'docker'
[done] √ Authentication successful (dscr.io)
Sending build context to Docker daemon  12.29kB
Step 1/6 : FROM golang:1.11
 ---> 901414995ecd
[...]
[info]   Image pushed to registry (dscr.io)
[done] √ Done processing image 'dscr.io/fabiankramm/devspace'
[info]   Deploying devspace-default with kubectl
deployment.extensions/devspace configured          
[done] √ Finished deploying devspace-default       
[info]   The Space is now reachable via ingress on this URL: https://fabian.devspace.host
[info]   Will now try to print the logs of a running devspace pod...
[info]   Printing logs of pod devspace-5d5cdb7554-lmxsg/default...
Hello World devspace!
```

# Troubleshooting 

If you experience problems during deploy or want to check if there are any issues within your deployed application devspace provides useful commands for you:
- `devspace analyze` analyzes the namespace and checks for warning events and failed pods / containers
- `devspace enter` open a terminal to a kubernetes pod (the same as running `kubectl exec ...`)
- `devspace logs` shows the logs of a devspace (the same as running `kubectl logs ...`)
- `devspace purge` delete the deployed application

See https://devspace.cloud/docs for more advanced documentation
