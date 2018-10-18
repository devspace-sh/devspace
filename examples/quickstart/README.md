# Quickstart example

This example shows you how to develop a small node express application with devspace.

# Step 0: Prerequisites

In order to use this example, make sure you have docker installed and a docker registry where you can push to (hub.docker.com, gcr.io etc.). Make sure you are logged in to the registry via `docker login`.  

Exchange the image name in `.devspace/config.yaml` under `images.default.name` with the image name you want to use. Do **not** add a tag to this image name, because this will be done at runtime automatically.  

## Optional: Use self hosted cluster (minikube, GKS etc.) instead of devspace-cloud

By default, this example will deploy to the devspace-cloud, a free managed kubernetes cluster. If you want to use your own cluster instead of the devspace-cloud as deployment target, make sure `kubectl` is configured correctly to access resources on the cluster. Then just erase the `cluster` section in the `.devspace/config.yaml` and devspace will use your current `kubectl` context as deployment target.

# Step 1: Start the devspace

To deploy the application to the devspace-cloud simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Building image 'fabian1991/quickstart' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  20.99kB
Step 1/7 : FROM node:8.11.4
 ---> 8198006b2b57
Step 2/7 : RUN mkdir /app
 ---> Using cache
 ---> 2064997c60c5
Step 3/7 : WORKDIR /app
 ---> Using cache
 ---> 6faeba82e3d7
Step 4/7 : COPY package.json .
 ---> Using cache
 ---> cb24ee28e9eb
Step 5/7 : RUN npm install
 ---> Using cache
 ---> a6ed836b6a83
Step 6/7 : COPY . .
 ---> f23d8c3c1c51
Step 7/7 : CMD ["npm", "start"]
 ---> Running in f1e2310d36e3
 ---> 98fbe8f46c11
Successfully built 98fbe8f46c11
Successfully tagged fabian1991/quickstart:dk0dqqO
The push refers to repository [docker.io/fabian1991/quickstart]
28bb9f0f148c: Pushed
090fce06793d: Layer already exists
e342c5b21403: Layer already exists
cbf8535e7a06: Layer already exists
be0fb77bfb1f: Layer already exists
63c810287aa2: Layer already exists
2793dc0607dd: Layer already exists
74800c25aa8c: Layer already exists
ba504a540674: Layer already exists
81101ce649d5: Layer already exists
daf45b2cad9a: Layer already exists
8c466bf4ca6f: Layer already exists
dk0dqqO: digest: sha256:5e043c3d366676331f4ffe6a9b6f38cbc08338c25ef47789060564d3304153a2 size: 2839
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/quickstart'
[INFO]   Deploying devspace-default with helm
[DONE] √ Deployed helm chart (Release revision: 2)
[DONE] √ Successfully deployed devspace-default
[DONE] √ Port forwarding started on 3000:3000
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/quickstart-kubectl <-> /app (Pod:e388779b2b49465855bb0322057a9fff/devspace-5b5f977b77-49cjt)
root@devspace-5b5f977b77-49cjt:/app#
```

The command built your Dockerfile and pushed it to the target docker registry. Afterwards, it created a new kubernetes namespace for you in the devspace-cloud and deployed the `kube/deployment.yaml` to that namespace. It also created a new kubectl context for you. If you want to access kubernetes resources via kubectl in the devspace-cloud you can simply change your kubectl context via `devspace up --switch-context`. Now you can check the running pods via `kubectl get po`.

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/quickstart` and `/app` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 3000 was forwarded to your local port 3000.  

# Step 2: Start developing

You can start the server now with `npm start` in the open terminal. Now navigate in your browser to `localhost:3000` and you should see the output 'Hello World!'.  

Change something in `index.js` locally and you should see something like this: 

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
