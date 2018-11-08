# Offline-Development example

This example shows you how to develop offline with a minikube setup. You have to run `devspace up` once while online to setup the environment correctly in your minikube cluster, but you can afterwards develop offline.

# Step 0: Prerequisites

In order to use this example, make sure you have a working minikube setup (you don't need an additional docker daemon, because devspace can use the internal minikube docker daemon). See [install minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) for more details.

# Step 1: Start DevSpace

This step has to be done once while being online. To deploy the application to minikube simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Create namespace test
[INFO]   Building image 'devspace' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  7.077MB
Step 1/8 : FROM node:8.11.4
 ---> 8198006b2b57
Step 2/8 : RUN mkdir /app
 ---> Using cache
 ---> be8130ce594c
Step 3/8 : WORKDIR /app
 ---> Using cache
 ---> a66cd053d094
Step 4/8 : COPY package.json .
 ---> Using cache
 ---> e84f0c80a89d
Step 5/8 : RUN npm install
 ---> Using cache
 ---> b972ec2c40f8
Step 6/8 : COPY . .
 ---> 867b7c08b0f4
Step 7/8 : EXPOSE 3000
 ---> Running in 50767081a2b8
 ---> e43d3516a051
Step 8/8 : CMD ["npm", "start"]
 ---> Running in 974c711c9c81
 ---> 687b0db6e4d5
Successfully built 687b0db6e4d5
Successfully tagged devspace:amodlCX
[INFO]   Skip image push for devspace
[DONE] √ Done building and pushing image 'devspace'
[INFO]   Deploying devspace-default with helm
[DONE] √ Tiller started
[DONE] √ Deployed helm chart (Release revision: 1)
[DONE] √ Finished deploying devspace-default
[DONE] √ Port forwarding started on 3000:3000
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/offline-development <-> /app (Pod: test/devspace-default-6446cb6b8c-c2l2q)
root@devspace-default-6446cb6b8c-c2l2q:/app#
```

The command deployed a tiller server and used the minikube docker daemon to build the dockerfile.  

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/offline-development` and `/app` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 3000 was forwarded to your local port 3000.  

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

You can also exit the terminal and reopen it with `devspace up` without the need of an internet connection.  
