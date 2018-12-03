# No Terminal example

This example shows you how to develop a small node express application with devspace without interacting in the remote container.

# Step 0: Prerequisites

In order to use this example, make sure you have docker installed and a docker registry where you can push to (hub.docker.com, gcr.io etc.). Make sure you are logged in to the registry via `docker login`.  

Exchange the image name in `.devspace/config.yaml` under `images.default.name` with the image name you want to use. Do the same thing in `kube/deployment.yaml` under `spec.template.spec.image`. Do **not** add a tag to those image names, because this will be done at runtime automatically.  

## Optional: Use self hosted cluster (minikube, GKE etc.) instead of devspace-cloud

By default, this example will deploy to the devspace-cloud, a free managed kubernetes cluster. If you want to use your own cluster instead of the devspace-cloud as deployment target, make sure `kubectl` is configured correctly to access resources on the cluster. Then just erase the `cluster` section in the `.devspace/config.yaml` and devspace will use your current `kubectl` context as deployment target.

# Step 1: Start the devspace

To deploy the application to the devspace-cloud simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Loading config .devspace/config.yaml with overwrite config .devspace/overwrite.yaml
[INFO]   Successfully logged into devspace-cloud
[INFO]   Building image 'fabian1991/devspace' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  214.5kB
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
 ---> d51aec0d55eb
Step 5/7 : RUN npm install
 ---> Using cache
 ---> 075f006f3d12
Step 6/7 : COPY . .
 ---> 963e993cfbff
Step 7/7 : CMD ["npm", "start"]
 ---> Running in 21caf6af8240
 ---> eb55b06d90a6
Successfully built eb55b06d90a6
Successfully tagged fabian1991/devspace:hOEPdbc
The push refers to repository [docker.io/fabian1991/devspace]
b6ca746752fa: Pushed
b2487fd09b6c: Layer already exists
1031aeb1d011: Layer already exists
cbf8535e7a06: Layer already exists
be0fb77bfb1f: Layer already exists
63c810287aa2: Layer already exists
2793dc0607dd: Layer already exists
74800c25aa8c: Layer already exists
ba504a540674: Layer already exists
81101ce649d5: Layer already exists
daf45b2cad9a: Layer already exists
8c466bf4ca6f: Layer already exists
hOEPdbc: digest: sha256:d8ad74fda8c8174cecf6cef18679d1c00f244372fb35da1392c72dce5fa16e00 size: 2840
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/devspace'
[INFO]   Deploying devspace-default with kubectl
deployment.extensions/devspace created
[DONE] √ Finished deploying devspace-default
[DONE] √ Port forwarding started on 3000:3000
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/no-terminal <-> /app (Pod: d98142a0dde341734e55d6f622b98c48/devspace-7c499c49c5-g72kj)
[INFO]   Your DevSpace is now reachable via ingress on this URL: http://fabiankramm-d98142a0.devspace-cloud.com
[INFO]   See https://devspace-cloud.com/domain-guide for more information
[INFO]   Will now try to attach to a running devspace pod...
[INFO]   Attaching to pod devspace-7c499c49c5-g72kj/default...
[nodemon] restarting due to changes...
[nodemon] restarting due to changes...
[nodemon] restarting due to changes...
[nodemon] restarting due to changes...
[nodemon] starting `node index.js`
Example app listening on port 3000!
```

The command built your Dockerfile and pushed it to the target docker registry. Afterwards, it created a new kubernetes namespace for you in the devspace-cloud and deployed the `kube/deployment.yaml` to that namespace. It also created a new kubectl context for you. You can check the running pods via `kubectl get po`.

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/no-terminal` and `/app` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 3000 was forwarded to your local port 3000. Also `devspace up` attached to the pod and prints the logs.  

# Step 2: Start developing

Navigate in your browser to `localhost:3000` and you should see the output 'Hello World!'.  

Change something in `index.js` locally and you should see something like this: 

```
[nodemon] restarting due to changes...
[nodemon] starting `node index.js`
Example app listening on port 3000!
```

Now just refresh your browser and you should see the changes immediately.  
