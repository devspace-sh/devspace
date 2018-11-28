# Reload instead of sync example

This example shows you how to develop a go application with devspace with reloading the build & deploy pipeline instead of hot reloading

# Step 0: Prerequisites

In order to use this example, make sure you have docker installed and a docker registry where you can push to (hub.docker.com, gcr.io etc.). Make sure you are logged in to the registry via `docker login`.  

Exchange the image name in `.devspace/config.yaml` under `images.default.name` with the image name you want to use. Do the same thing in `kube/deployment.yaml` under `spec.template.spec.image`. Do **not** add a tag to those image names, because this will be done at runtime automatically.  

## Optional: Use self hosted cluster (minikube, GKE etc.) instead of devspace-cloud

If you want to use your own cluster instead of the devspace-cloud as deployment target, make sure `kubectl` is configured correctly to access your cluster. Then just erase the `cluster` section in the `.devspace/config.yaml` and devspace will use your current `kubectl` context as deployment target.  

# Step 1: Start the devspace

To deploy the application simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Loading config .devspace/config.yaml with overwrite config .devspace/overwrite.yaml
[INFO]   Successfully logged into devspace-cloud
[INFO]   Building image 'fabian1991/devspace' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  14.34kB
Step 1/6 : FROM golang:1.11
 ---> fb7a47d8605b
Step 2/6 : RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
 ---> Using cache
 ---> 9592048b1d0b
Step 3/6 : ADD . $GOPATH/src/app
 ---> Using cache
 ---> c46f025a11b5
Step 4/6 : RUN cd $GOPATH/src/app && go get ./... && go build . && cd /app
 ---> Using cache
 ---> c1ad1a294a4f
Step 5/6 : WORKDIR /app
 ---> Using cache
 ---> 35608cd6e592
Step 6/6 : CMD ["$GOPATH/src/app/app"]
 ---> Using cache
 ---> af6e428fdcef
Successfully built af6e428fdcef
Successfully tagged fabian1991/devspace:mlSwh8F
The push refers to repository [docker.io/fabian1991/devspace]
36ffc46d7733: Pushed
730329ceeb12: Pushed
18f7b628e662: Pushed
cbc3342714a6: Layer already exists
cf2043c0e99a: Layer already exists
84a5f96d6621: Layer already exists
ab016c9ea8f8: Layer already exists
2eb1c9bfc5ea: Layer already exists
0b703c74a09c: Layer already exists
b28ef0b6fef8: Layer already exists
mlSwh8F: digest: sha256:a72aaa06264f9be86da167a78d00f7d1c3a4468f4916ea7dbe26dcdc28f6a697 size: 2422
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/devspace'
[INFO]   Deploying devspace-default with helm
[DONE] √ Tiller started
[DONE] √ Deployed helm chart (Release revision: 1)
[DONE] √ Finished deploying devspace-default
[INFO]   Your DevSpace is now reachable via ingress on this URL: http://fabiankramm-d98142a0.devspace-cloud.com
[INFO]   See https://devspace-cloud.com/domain-guide for more information
Fabians-MBP:reload-instead-of-sync fabiankramm$ devspace up
[INFO]   Loading config .devspace/config.yaml with overwrite config .devspace/overwrite.yaml
[INFO]   Successfully logged into devspace-cloud
[INFO]   Building image 'fabian1991/devspace' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  10.75kB
Step 1/6 : FROM golang:1.11
 ---> fb7a47d8605b
Step 2/6 : RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
 ---> Using cache
 ---> 9592048b1d0b
Step 3/6 : ADD . $GOPATH/src/app
 ---> a1ee1b4ae832
Step 4/6 : RUN cd $GOPATH/src/app && cd /app
 ---> Running in 72c82ac8c356
 ---> c6895e03655c
Step 5/6 : WORKDIR /app
 ---> Running in 24f6d88d1982
 ---> 49f341db3a9f
Step 6/6 : CMD ["go", "run", "main.go"]
 ---> Running in 6b39765f9e20
 ---> fd3c715ed7f4
Successfully built fd3c715ed7f4
Successfully tagged fabian1991/devspace:aXiK5GW
The push refers to repository [docker.io/fabian1991/devspace]
6791cab63553: Pushed
18f7b628e662: Layer already exists
cbc3342714a6: Layer already exists
cf2043c0e99a: Layer already exists
84a5f96d6621: Layer already exists
ab016c9ea8f8: Layer already exists
2eb1c9bfc5ea: Layer already exists
0b703c74a09c: Layer already exists
b28ef0b6fef8: Layer already exists
aXiK5GW: digest: sha256:c19c53b9f72de998ed8de81bc6591aef91bc23e182ca807d3430ead94ffac3d0 size: 2211
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/devspace'
[INFO]   Deploying devspace-default with kubectl
deployment.extensions/devspace created
[DONE] √ Finished deploying devspace-default
[INFO]   Will now try to attach to a running devspace pod...
[INFO]   Attaching to pod devspace-66c788c9f8-lrtv2/default...
Hello World!
Hello World!
```

The command built your Dockerfile and pushed it to the target docker registry. Afterwards, it created a new kubernetes namespace for you and deployed the `kube/deployment.yaml` to that namespace. It also created a new kubectl context for you. If you want to access kubernetes resources via kubectl in the devspace-cloud you can simply change your kubectl context via `devspace up --switch-context`. Now you can check the running pods via `kubectl get po`.

Furthermore it watches all files in the example for changes, which will trigger on change an automatic rebuild & redeploy.

# Step 2: Start developing

You should see a "Hello World!" printed every second in the console. Try changing `main.go` so that it prints "Hello DevSpace!" instead of "Hello World!". In the console window a similar output should be shown:

```
[INFO]   Change detected, will reload in 2 seconds
Hello World!
Hello World!
[INFO]   Building image 'fabian1991/devspace' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  10.75kB
Step 1/6 : FROM golang:1.11
 ---> fb7a47d8605b
Step 2/6 : RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
 ---> Using cache
 ---> 9592048b1d0b
Step 3/6 : ADD . $GOPATH/src/app
 ---> 52e10ca4c98e
Step 4/6 : RUN cd $GOPATH/src/app && cd /app
 ---> Running in d14db7b806d5
 ---> 2cfa70fd538d
Step 5/6 : WORKDIR /app
 ---> Running in 29a9f491dff8
 ---> 4ba94b76c2fd
Step 6/6 : CMD ["go", "run", "main.go"]
 ---> Running in a479f87f252c
 ---> 98c5aac29268
Successfully built 98c5aac29268
Successfully tagged fabian1991/devspace:9LCpwdd
The push refers to repository [docker.io/fabian1991/devspace]
d3666866ac6e: Pushed
18f7b628e662: Layer already exists
cbc3342714a6: Layer already exists
cf2043c0e99a: Layer already exists
84a5f96d6621: Layer already exists
ab016c9ea8f8: Layer already exists
2eb1c9bfc5ea: Layer already exists
0b703c74a09c: Layer already exists
b28ef0b6fef8: Layer already exists
9LCpwdd: digest: sha256:f6bba6e00c42d099d08d415d794ae3202f64bd70589a4e189e1062890e894440 size: 2211
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/devspace'
[INFO]   Deploying devspace-default with kubectl
deployment.extensions/devspace configured
[DONE] √ Finished deploying devspace-default
[INFO]   Will now try to attach to a running devspace pod...
[INFO]   Attaching to pod devspace-859d5ccbd7-x77hc/default...
Hello DevSpace!
Hello DevSpace!
Hello DevSpace!
```

Now you can just change any files and devspace will rebuild and redeploy your changes immediately.
