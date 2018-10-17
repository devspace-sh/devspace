# Quickstart kubectl example

This example shows you how to develop a small node express application with devspace and devspace-cloud.

# Step 0: Prerequisites

In order to use this example, make sure you have docker installed and a docker registry where you can push to (hub.docker.com, gcr.io etc.). Make sure you are logged in to the registry via `docker login`.  

Exchange the image name in `.devspace/config.yaml` under `images.default.name` with the image name you want to use. Do the same thing in `kube/deployment.yaml` under `spec.template.spec.image`. Do **not** add a tag to those image names, because this will be done at runtime automatically.  

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
[INFO]   Deploying devspace-default with kubectl
deployment.extensions/devspace created
[DONE] √ Successfully deployed devspace-default
[DONE] √ Port forwarding started on 3000:3000
[DONE] √ Sync started on /Users/fabiankramm/Programmieren/go-workspace/src/github.com/covexo/devspace/examples/quickstart-kubectl <-> /app (Pod:e388779b2b49465855bb0322057a9fff/devspace-5b5f977b77-49cjt)
root@devspace-5b5f977b77-49cjt:/app#
```

The command created a new kubernetes namespace for you in the devspace-cloud and deployed the `kube/deployment.yaml` to that namespace. It also created a new kubectl context for you. If you want to access kubernetes resources via kubectl in the devspace-cloud you can simply change your kubectl context via `kubectl config use-context $$devspace-context-name$$`. You can find the context name in `.devspace/config.yaml` under `cluster.kubeContext`.  

# Step 2: Start developing

You can start the server now with `npm start` in the open terminal. Now navigate in your browser to `localhost:3000` and you should see the output 'Hello World!'.  

You can easily change any code within the `index.js` and restart the server with `npm start` and you should see the changes immediately, without the need of rebuilding the docker file or redeploying the chart.  
