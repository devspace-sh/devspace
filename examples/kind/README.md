# Using DevSpace with KIND

[kind](https://kind.sigs.k8s.io) is a tool for running local Kubernetes clusters using Docker container “nodes”.

## Prerequisite:

- [Docker](https://docs.docker.com/get-docker/)
- [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [DevSpace](https://devspace.sh/cli/docs/getting-started/installation)


## Steps:

### Create a cluster and local registry using KIND

To create a cluster and registry to be used by the KIND cluster please follow the [documentation](https://kind.sigs.k8s.io/docs/user/local-registry/).

### Clone this project

```sh
git clone git@github.com:loft-sh/devspace.git

cd examples/kind
```

### Create new namespace in KIND cluster

```sh
kubectl create ns test-1
```

### Start developing application with devspace

```sh
devspace dev -n test-1
```

You'll see output similar to this:

```sh
[info] Using namespace 'test-1'
[info] Using kube context 'kind-kind'
[info] Building image 'localhost:5000/app:NsZqtco' with engine 'docker'
[done] √ Authentication successful (localhost:5000)
Sending build context to Docker daemon 60.42kB
Step 1/7 : FROM node:13.14-alpine
---> 6ebf8a36c218
Step 2/7 : RUN mkdir /app
---> Using cache
---> 6bcbf7f20618
Step 3/7 : WORKDIR /app
---> Using cache
---> 62101e1a2a94
Step 4/7 : COPY package.json .
---> Using cache
---> deda2919c124
Step 5/7 : RUN npm install
---> Using cache
---> 5c5d63f8d6a3
Step 6/7 : COPY . .
---> 052c7e5d908b
Step 7/7 : CMD ["npm", "start"]
---> Running in 06038f235b19
---> 5591da86fabf
Successfully built 5591da86fabf
Successfully tagged localhost:5000/app:NsZqtco
The push refers to repository [localhost:5000/app]
8850e2de6704: Pushed
f4da1b8b450d: Layer already exists
c2b2d07a43bf: Layer already exists
1c60dde6f5e0: Layer already exists
0a454283a2a4: Layer already exists
0251486b80af: Layer already exists
8940ae814902: Layer already exists
678a0785e7d2: Layer already exists
NsZqtco: digest: sha256:96550aba92e3df23bc0f8b6f70fbba3e573a11d3311cd05342f02b92f211b6fa size: 1992
[info] Image pushed to registry (localhost:5000)
[done] √ Done processing image 'localhost:5000/app'
[info] Execute 'helm list --namespace test-1 --output json --kube-context kind-kind'
[info] Execute 'helm upgrade quickstart /Users/pratikjagrut/.devspace/component-chart/component-chart-0.8.1.tgz --namespace test-1 --values /var/folders/9c/mv8m0y2j5_13zyd93yxwl71m0000gn/T/667828839 --install --kube-context kind-kind'
[info] Execute 'helm list --namespace test-1 --output json --kube-context kind-kind'
[done] √ Deployed helm chart (Release revision: 1)
[done] √ Successfully deployed quickstart with helm

#########################################################
[info] DevSpace UI available at: http://localhost:8090
#########################################################

[done] √ Port forwarding started on 3000:3000 (test-1/quickstart-55687985f-h9nqm)
[0:sync] Waiting for pods...
[0:sync] Starting sync...
[0:sync] Inject devspacehelper into pod test-1/quickstart-55687985f-h9nqm
[0:sync] Start syncing
[0:sync] Sync started on /Users/pratikjagrut/work/github.com/loft-sh/devspace/examples/kind <-> . (Pod: test-1/quickstart-55687985f-h9nqm)
[0:sync] Waiting for initial sync to complete
[0:sync] Helper - Use inotify as watching method in container
[0:sync] Downstream - Initial sync completed
[0:sync] Upstream - Upload File 'Dockerfile'
[0:sync] Upstream - Upload File 'devspace.yaml'
[0:sync] Upstream - Upload 2 create change(s) (Uncompressed ~0.57 KB)
[0:sync] Upstream - Successfully processed 2 change(s)
[0:sync] Upstream - Initial sync completed
[info] Starting log streaming
[quickstart] Start streaming logs for test-1/quickstart-55687985f-h9nqm/container-0
[quickstart]
[quickstart] > node-js-sample@0.0.1 start /app
[quickstart] > nodemon index.js
[quickstart]
[quickstart] [nodemon] 2.0.12
[quickstart] [nodemon] to restart at any time, enter `rs`
[quickstart] [nodemon] watching path(s): *.*
[quickstart] [nodemon] watching extensions: js,mjs,json
[quickstart] [nodemon] starting `node index.js`
[quickstart] Example app listening on port 3000!
```

**Goto browser and hit http://localhost:3000**

You'll see

```sh
Hello World!
```

### Make changes to source code and see the devspace magic

Change line `res.send('Hello World!');` to `res.send("Hello K8S, it's DevSpace!");` and save the file.

You'll see changes in the terminal:

```sh
[0:sync] Upstream - Upload File 'index.js'
[0:sync] Upstream - Upload 1 create change(s) (Uncompressed ~0.22 KB)
[0:sync] Upstream - Successfully processed 1 change(s)
[quickstart] [nodemon] restarting due to changes...
[quickstart] [nodemon] starting `node index.js`
[quickstart] Example app listening on port 3000!
```

And now reload the browser.

You'll see

```sh
Hello K8S, it's DevSpace!
```

## Important notes

### Make sure to create cluster and registry as shown in step one

### Don't skip-push

In this scenario, devspace is building an image outside the cluster using docker and it is stored in the docker-local registry. To use this image in the KIND cluster, we need to make sure this image is pushed to the KIND registry which we created in the first step. For this make sure to change the image name `username/app` to `KIND-registry-host/app`. In this case KIND registry is hosted at `localhost:5000` so our image name will be `localhost:5000/app`.

*The hostname and port are decided while creating the KIND cluster and registry.*


## Troubleshooting

It is possible sometimes `containerd` to reject pulling images from non-secure registries.
In this case, we can instruct containerd to trust the non-secure local registry when pulling.

Just replace the containerd configuration with below in the `create-a-cluster-and-registry script`.

```sh
containerdConfigPatches:
- |-
 [plugins."io.containerd.grpc.v1.cri".registry]
 [plugins."io.containerd.grpc.v1.cri".registry.configs]
 [plugins."io.containerd.grpc.v1.cri".registry.configs."${reg_name}:${reg_port}"]
 [plugins."io.containerd.grpc.v1.cri".registry.configs."${reg_name}:${reg_port}".tls]
 insecure_skip_verify = true
 [plugins."io.containerd.grpc.v1.cri".registry.mirrors."${reg_name}:${reg_port}"]
 endpoint = ["http://${reg_name}:${reg_port}"]
EOF
```

The sample script is available [here](kind-with-registry-non-secure.sh).
