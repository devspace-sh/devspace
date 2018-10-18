# Microservices example

This example shows you how to develop two microservices in a single repo. The example consists of a small node webserver that makes a http request to a php webserver to retrieve some data.

# Step 0: Prerequisites

In order to use this example, make sure you have docker installed and a docker registry where you can push to (hub.docker.com, gcr.io etc.). Make sure you are logged in to the registry via `docker login`.  

Exchange the image name in `.devspace/config.yaml` under `images.php.name` and `images.node.name` with the image name you want to use. Do the same thing in `node/kube/deployment.yaml` under `spec.template.spec.image`.  Do **not** add a tag to these image names, because this will be done at runtime automatically.  

## Optional: Use self hosted cluster (minikube, GKS etc.) instead of devspace-cloud

If you want to use your own cluster instead of the devspace-cloud as deployment target, make sure `kubectl` is configured correctly to access your cluster. Then just erase the `cluster` section in the `.devspace/config.yaml` and devspace will use your current `kubectl` context as deployment target.

# Step 1: Start the devspace

To deploy the application simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Building image 'fabian1991/php' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  8.704kB
Step 1/6 : FROM php:7.1-apache-stretch
 ---> 93e6fb4b13e1
[...]
Step 6/6 : RUN usermod -u 1000 www-data;     a2enmod rewrite;     chown -R www-data:www-data /var/www/html
 ---> Using cache
 ---> 4b6e6f1150d3
Successfully built 4b6e6f1150d3
Successfully tagged fabian1991/php:Tsgwdi8
The push refers to repository [docker.io/fabian1991/php]
bfee725f50e2: Layer already exists
[...]
237472299760: Layer already exists
Tsgwdi8: digest: sha256:9e85195e0793af26e15181cb771d93acdc1ad40e3126193acd26eb4eb3765a03 size: 3867
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/php'
[INFO]   Building image 'fabian1991/node' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon    108kB
Step 1/7 : FROM node:8.11.4
 ---> 8198006b2b57
[...]
Step 7/7 : CMD ["npm", "start"]
 ---> Using cache
 ---> ea42e151ef28
Successfully built ea42e151ef28
Successfully tagged fabian1991/node:fq2KN6i
The push refers to repository [docker.io/fabian1991/node]
d3f119d48426: Layer already exists
[...]
8c466bf4ca6f: Layer already exists
fq2KN6i: digest: sha256:0b7e9393b3300f2f2cb54db442417db16ae48bf66f0061b8d36cdcc7cc84d6c0 size: 2841
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/node'
[INFO]   Deploying devspace-node with kubectl
deployment.extensions/devspace configured
[DONE] √ Successfully deployed devspace-node
[INFO]   Deploying devspace-php with helm
[DONE] √ Tiller started
[DONE] √ Deployed helm chart (Release revision: 1)
[DONE] √ Successfully deployed devspace-php
[DONE] √ Port forwarding started on 3000:3000
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/microservices/node <-> /app (Pod:test/devspace-7ffbf854ff-hk4jq)
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/microservices/php <-> /var/www/html (Pod: test/devspace-php-7f9b876786-hcwxz)
root@devspace-7ffbf854ff-hk4jq:/app#
```

The command built two docker images: One docker image for the node application in `node/Dockerfile` and one for the php application in `php/Dockerfile`. Then the images were pushed to the docker registry. Afterwards, the deployment.yaml in `node/kube/deployment.yaml` was deployed with kubectl and the chart in `php/chart` was deployed via helm.  

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/microservices/node` and the `/app` folder within the node container. Also the folder `/go-workspace/src/github.com/covexo/devspace/examples/microservices/php` and the `/var/www/html` are synchronized. Whenever you change a file in either of those folders the change will be synchronized. In addition the node container port 3000 was forwarded to your local port 3000.  

# Step 2: Start developing

A terminal to the node container should been automatically opened. You can start the server now with `npm start` in the open terminal. Now navigate in your browser to `localhost:3000` and you should see the output 'PHP say's Hello World!'. The node server just did a request to the php server in the background.  

Now try to change the `php/index.php` and alter the message. Simply refresh your browser and your changes should be visible!  

You can also change something in `node/index.js` locally and you should see something like this: 

```
[nodemon] 1.18.4
[nodemon] to restart at any time, enter `rs`
[nodemon] watching: *.*
[nodemon] starting `node index.js`
Example app listening on port 3000!
[nodemon] restarting due to changes...
[nodemon] starting `node index.js`
Example app listening on port 3000!
```

Now just refresh your browser and you should see the changes immediately. 
