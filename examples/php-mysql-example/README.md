# Php Mysql example

This example shows you how to develop and deploy a small php and mysql application with devspace. For a more detailed documentation, take a look at https://devspace.cloud/docs

# Step 0: Prerequisites

In order to get things ready do the following:
1. Install docker
2. Run `devspace create space quickstart` to create a new kubernetes namespace in the devspace.cloud (if you want to use your own cluster just skip this step and devspace will use the current kubectl context, you probably also want to exchange the image name in `devspace.yaml`)

# Step 1: Develop the application

1. Run `devspace dev` to start the application in development mode

The command does several things in this order:
- Build the docker image
- Setup tiller in the namespace and create needed image pull secrets
- Deploy the components (an easy to use helm chart), you can also deploy your custom helm chart, kustomize or regular kubectl manifests
- Start port forwarding the remote port 80 to local port 8080 
- Start syncing all files in the php-mysql-example folder with the remote container in /var/www/html
- Open a terminal to the remote pod

You should see the following output:
```
[info]   Loaded config from devspace.yaml
[info]   Using space fabian                       
[info]   Building image 'dscr.io/fabiankramm/devspace' with engine 'docker'
[done] √ Authentication successful (dscr.io)
Sending build context to Docker daemon  9.031kB
Step 1/8 : FROM php:7.1-apache-stretch
 ---> 8198006b2b57
[...]
[info]   Image pushed to registry (dscr.io)                         
[info]   Deploying devspace-app with helm                                                          
[done] √ Finished deploying devspace-app
[done] √ Port forwarding started on 8080:80             
[done] √ Sync started on /github.com/devspace-cloud/devspace/examples/php-mysql-example <-> /app (Pod: d4c1654922db400f612a027283b50001/default-74d58cbc59-9j4mj)
root@default-7c4dcdfc4-m867d:/var/www/html#
```
2. Go to `localhost:8080` to see the output of the webserver 
4. Change something in the `index.php`
5. Refresh the browser to see the changes applied.

# Step 2: Deploy the application

Deploying the application is the same as developing it, but instead of `devspace dev` you run `devspace deploy`. The deploy command does not start any of the developing services (port-forwarding, sync and terminal) and just deploys the application.

# Troubleshooting 

If you experience problems during deploy or want to check if there are any issues within your deployed application devspace provides useful commands for you:
- `devspace analyze` analyzes the namespace and checks for warning events and failed pods / containers
- `devspace enter` open a terminal to a kubernetes pod (the same as running `kubectl exec ...`)
- `devspace logs` shows the logs of a devspace (the same as running `kubectl logs ...`)
- `devspace purge` delete the deployed application
- `devspace open` to create an ingress and open the application in the browser

See https://devspace.cloud/docs for more advanced documentation
