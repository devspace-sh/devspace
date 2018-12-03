# Kaniko example

This example shows how kaniko can be used instead of docker to build and push an docker image directly inside the cluster.   

# Step 0: Prerequisites

In order to use this example, make sure you have a working minikube setup (you don't need an additional docker daemon or docker registry, because devspace can use the internal minikube docker daemon and will deploy a registry for you). See [install minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) for more details.

# Step 1: Start the devspace

To deploy the application to minikube simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Create namespace test
[DONE] √ Tiller started
[DONE] √ Internal registry started
[INFO]   Building image 'devspace' with engine 'kaniko'
[DONE] √ Authentication successful (10.107.78.184:5000)
[DONE] √ Kaniko build pod started
[DONE] √ Uploaded files to container
[INFO]   build >>> WARN[0001] Error while retrieving image from cache: getting image from path: open /cache/sha256:df8db4a3a7dee9782e0f1bdcc9d676bc2de0dc1d2dc2952d9b9b3718445b1455: no such file or directory
[INFO]   build >>> INFO[0001] Downloading base image golang:1.11
[INFO]   build >>> 2018/10/18 11:54:05 No matching credentials were found, falling back on anonymous
[INFO]   build >>> INFO[0002] Executing 0 build triggers
[INFO]   build >>> INFO[0002] Extracting layer 0
[INFO]   build >>> INFO[0014] Extracting layer 1
[INFO]   build >>> INFO[0017] Extracting layer 2
[INFO]   build >>> INFO[0018] Extracting layer 3
[INFO]   build >>> INFO[0029] Extracting layer 4
[INFO]   build >>> INFO[0042] Extracting layer 5
[INFO]   build >>> INFO[0076] Extracting layer 6
[INFO]   build >>> INFO[0076] Taking snapshot of full filesystem...
[INFO]   build >>> INFO[0090] RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
[INFO]   build >>> INFO[0090] Checking for cached layer 10.107.78.184:5000/devspace/cache:f141885fc82d849f3eba2d72bf74ce9842b3f9874ef5b003dd9b846726ee46b4...
[INFO]   build >>> INFO[0090] No cached layer found, executing command...
[INFO]   build >>> INFO[0090] cmd: /bin/sh
[INFO]   build >>> INFO[0090] args: [-c mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app]
[INFO]   build >>> INFO[0090] Using files from context: [/src]
[INFO]   build >>> INFO[0090] ADD . $GOPATH/src/app
[INFO]   build >>> INFO[0090] RUN cd $GOPATH/src/app && go get ./... && go build . && cd /app
[INFO]   build >>> INFO[0090] Checking for cached layer 10.107.78.184:5000/devspace/cache:3ade7ab92e4a6e6d8d57c98137a987ed85d67ba1446c2d92be842d14dd44ea67...
[INFO]   build >>> INFO[0090] No cached layer found, executing command...
[INFO]   build >>> INFO[0090] cmd: /bin/sh
[INFO]   build >>> INFO[0090] args: [-c cd $GOPATH/src/app && go get ./... && go build . && cd /app]
[INFO]   build >>> INFO[0093] WORKDIR /app
[INFO]   build >>> INFO[0093] cmd: workdir
[INFO]   build >>> INFO[0093] Changed working directory to /app
[INFO]   build >>> INFO[0093] CMD ["$GOPATH/src/app/app"]
[INFO]   build >>> INFO[0093] Taking snapshot of full filesystem...
[INFO]   build >>> 2018/10/18 11:55:48 pushed blob sha256:af3d9268d1a6b25f664130670edb460efcb7dd6e22f58efcc6cef2714c7a7efe
[INFO]   build >>> 2018/10/18 11:55:49 pushed blob sha256:202760eb4a0043cd84cd9971c47052617855ff653abec8ae479e89d369afd500
[INFO]   build >>> 2018/10/18 11:55:49 pushed blob sha256:8e9d103264e8425af20c8ae84535d73008e5accc340f95e9e3e155132053bae4
[INFO]   build >>> 2018/10/18 11:55:55 pushed blob sha256:e5c3f8c317dc30af45021092a3d76f16ba7aa1ee5f18fec742c84d4960818580
[INFO]   build >>> 2018/10/18 11:56:02 pushed blob sha256:193a6306c92af328dbd41bbbd3200a2c90802624cccfe5725223324428110d7f
[INFO]   build >>> 2018/10/18 11:56:35 pushed blob sha256:bc9ab73e5b14b9fbd3687a4d8c1f1360533d6ee9ffc3f5ecc6630794b40257b7
[INFO]   build >>> 2018/10/18 11:56:36 pushed blob sha256:a587a86c9dcb9df6584180042becf21e36ecd8b460a761711227b4b06889a005
[INFO]   build >>> 2018/10/18 11:56:41 pushed blob sha256:1bc310ac474b880a5e4aeec02e6423d1304d137f1a8990074cb3ac6386a0b654
[INFO]   build >>> 2018/10/18 11:56:59 pushed blob sha256:997731689cfbc58c8e74f2a20079338ce66965a40b21f27169b3d5a45ab61cbd
[INFO]   build >>> 2018/10/18 11:56:59 10.107.78.184:5000/devspace:JMNVk87: digest: sha256:6f9cc913b59a167050c9d65deb9677870880b27ccd842155980673f11a4cc205 size: 1578
[DONE] √ Done building image
[INFO]   Image pushed to registry (10.107.78.184:5000)
[DONE] √ Done building and pushing image '10.107.78.184:5000/devspace'
[INFO]   Deploying devspace-default with helm
[DONE] √ Deployed helm chart (Release revision: 1)
[DONE] √ Successfully deployed devspace-default
[DONE] √ Port forwarding started on 8080:8080
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/kaniko-minikube <-> /app (Pod: test/devspace-default-749f45ddcc-vgp4z)
root@devspace-default-749f45ddcc-vgp4z:/go/src/app#
```

The command created a test namespace, deployed a tiller server and internal registry and used a kaniko build pod to build the dockerfile.  

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/kaniko-minikube` and `/app` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 8080 was forwarded to your local port 8080.  

# Step 2: Start developing

You can start the server now with `go run main.go` in the open terminal. Now navigate in your browser to `localhost:8080` and you should see the output 'Hello World!'.  

Change something in `main.go` locally and re-run `go run main.go`. Now just refresh your browser and you should see the changes immediately.  
