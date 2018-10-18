# Offline-Development example

This example shows you how to develop offline with a minikube setup. You have to run `devspace up` once while online to setup the environment correctly in your minikube cluster, but you can afterwards develop offline.

# Step 0: Prerequisites

In order to use this example, make sure you have a working minikube setup (you don't need an additional docker daemon, because devspace can use the internal minikube docker daemon). See [install minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) for more details.

# Step 1: Start DevSpace

This step has to be done once while being online. To deploy the application to minikube simply run `devspace up`. The output of the command should look similar to this: 

```
[DONE] √ Tiller started
[DONE] √ Internal registry started
[INFO]   Building image 'devspace' with engine 'docker'
[DONE] √ Authentication successful (10.102.46.101:5000)
Sending build context to Docker daemon  6.144kB
Step 1/7 : FROM node:8.11.4
 ---> 8198006b2b57
Step 2/7 : RUN mkdir /app
 ---> Using cache
 ---> 1b6632b2da50
Step 3/7 : WORKDIR /app
 ---> Using cache
 ---> 20b4e5a1df9b
Step 4/7 : COPY package.json .
 ---> ee7f6e81e51d
Step 5/7 : RUN npm install
 ---> Running in e6ef4c082b0c
npm notice created a lockfile as package-lock.json. You should commit this file.
npm WARN node-js-sample@0.0.1 No repository field.

added 48 packages in 1.946s
 ---> ff5be7678a3d
Step 6/7 : COPY . .
 ---> b20037e9623f
Step 7/7 : CMD ["npm", "start"]
 ---> Running in abd0c9294587
 ---> f8d49e9378ff
Successfully built f8d49e9378ff
Successfully tagged 10.102.46.101:5000/devspace:oswQSfh
The push refers to repository [10.102.46.101:5000/devspace]
e4f99e03005a: Pushed
c9515cc05f90: Pushed
9ad0fa9ab2ad: Pushed
10959d10898a: Pushed
be0fb77bfb1f: Pushed
63c810287aa2: Pushed
2793dc0607dd: Pushed
74800c25aa8c: Pushed
ba504a540674: Pushed
81101ce649d5: Pushed
daf45b2cad9a: Pushed
8c466bf4ca6f: Pushed
oswQSfh: digest: sha256:af6f6f701136149dc303aad52c124ab8913015e8125ae68994e06325c327cc2e size: 2839
[INFO]   Image pushed to registry (10.102.46.101:5000)
[DONE] √ Done building and pushing image 'devspace'
[INFO]   Deploying devspace-default with helm
[DONE] √ Deployed helm chart (Release revision: 1)
[DONE] √ Successfully deployed devspace-default
[DONE] √ Port forwarding started on 3000:3000
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/minikube <-> /app (Pod: test/devspace-default-6446cb6b8c-c2l2q)
root@devspace-default-6446cb6b8c-c2l2q:/app#
```

The command deployed a tiller server, internal registry and used the minikube docker daemon to build the dockerfile.  

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/offline-development` and `/app` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 3000 was forwarded to your local port 3000.  

# Step 2: Start developing

You can start the server now with `npm start` in the open terminal. Now navigate in your browser to `localhost:3000` and you should see the output 'Hello World!'.  

Change something in `index.js` and you should see something like this: 

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

You can also exit the terminal and reopen it with `devspace up` without the need of an internet connection.  
