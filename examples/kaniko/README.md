# Kaniko example

This example shows how kaniko can be used instead of docker to build and push an docker image directly inside the cluster.   

# Step 0: Prerequisites

In order for this example to work you need access to a docker registry, where you can push images to (e.g. hub.docker.com, gcr.io etc.). There are three options how you can push images to registries with devspace. 

## Option 1: Use docker credentials store
If you have docker installed, devspace can take the required auth information directly out of the docker credentials store and will create the needed secret for you in the target cluster automatically. Make sure you are logged in the registry with `docker login`.  

## Option 2: Provide auth information yourself
As a second option you can provide your credentials directly in the config.yaml and devspace cli will create a pull secret for you automatically. See example below:

```yaml
images:
  default:
    build:
      kaniko:
        cache: true
    # Don't prefix image name with registry url 
    name: name/devspace
    registry: myRegistry
registries:
  myRegistry:
    # Registry url here
    url: gcr.io 
    auth:
      username: my-user
      password: my-password
```

devspace will then automatically create a secret for you which kaniko can use to push to that registry.  

## Option 3: Provide kaniko pull secret yourself
As a third option you can provide the pullSecret to use for kaniko yourself. Make sure the pull secret has the following form:

```yaml
apiVersion: v1
kind: Secret
data:
  # You need to specify the .dockerconfigjson (which will be mounted in the executor pod in /root/.docker/config.json) encoded in base64 e.g.: 
  # {
  #		"auths": {
  #			"<registryUrl>": {
  #				"auth": "<base64Encoded(user:password/token)>",
  #				"email": "<myemail@test.de>"
  #			}
  #		}
  #	}
  .dockerconfigjson: <BASE64EncodedDockerConfigJson>
```

Now specify the pullsecret name as the pull secret to use for kaniko in the .devspace/config:

```yaml
images:
  default:
    build:
      kaniko:
        cache: true
    name: registryName/name/devspace
    pullSecret: myPullSecretName
```

## Optional: Use self hosted cluster (minikube, GKE etc.) instead of devspace-cloud

If you want to use your own cluster instead of the devspace-cloud as deployment target, make sure `kubectl` is configured correctly to access the target cluster. Then just erase the `cluster` section in the `.devspace/config.yaml` and devspace will use your current kubectl context as deployment target.  

# Step 1: Start the devspace

To deploy the application to the target cluster simply run `devspace up`. The output of the command should look similar to this: 

```
[INFO]   Building image 'fabian1991/devspace' with engine 'kaniko'
[DONE] √ Authentication successful (hub.docker.com)
[DONE] √ Kaniko build pod started
[DONE] √ Uploaded files to container
[INFO]   build >>> WARN[0001] Error while retrieving image from cache: getting image from path: open /cache/sha256:df8db4a3a7dee9782e0f1bdcc9d676bc2de0dc1d2dc2952d9b9b3718445b1455: no such file or directory
[INFO]   build >>> INFO[0001] Downloading base image golang:1.11
[INFO]   build >>> ERROR: logging before flag.Parse: E1018 09:27:56.335503      34 metadata.go:142] while reading 'google-dockercfg' metadata: http status code: 404 while fetching url http://metadata.google.internal./computeMetadata/v1/instance/attributes/google-dockercfg
[INFO]   build >>> ERROR: logging before flag.Parse: E1018 09:27:56.337941      34 metadata.go:159] while reading 'google-dockercfg-url' metadata: http status code: 404 while fetching url http://metadata.google.internal./computeMetadata/v1/instance/attributes/google-dockercfg-url
[INFO]   build >>> INFO[0002] Executing 0 build triggers
[INFO]   build >>> INFO[0002] Extracting layer 0
[INFO]   build >>> INFO[0017] Extracting layer 1
[INFO]   build >>> INFO[0020] Extracting layer 2
[INFO]   build >>> INFO[0021] Extracting layer 3
[INFO]   build >>> INFO[0040] Extracting layer 4
[INFO]   build >>> INFO[0060] Extracting layer 5
[INFO]   build >>> INFO[0103] Extracting layer 6
[INFO]   build >>> INFO[0104] Taking snapshot of full filesystem...
[INFO]   build >>> INFO[0503] RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
[INFO]   build >>> INFO[0503] Checking for cached layer index.docker.io/fabian1991/devspace/cache:f141885fc82d849f3eba2d72bf74ce9842b3f9874ef5b003dd9b846726ee46b4...
[INFO]   build >>> ERROR: logging before flag.Parse: E1018 09:36:18.645323      34 metadata.go:142] while reading 'google-dockercfg' metadata: http status code: 404 while fetching url http://metadata.google.internal./computeMetadata/v1/instance/attributes/google-dockercfg
[INFO]   build >>> ERROR: logging before flag.Parse: E1018 09:36:18.648540      34 metadata.go:159] while reading 'google-dockercfg-url' metadata: http status code: 404 while fetching url http://metadata.google.internal./computeMetadata/v1/instance/attributes/google-dockercfg-url
[INFO]   build >>> INFO[0505] No cached layer found, executing command...
[INFO]   build >>> INFO[0505] cmd: /bin/sh
[INFO]   build >>> INFO[0505] args: [-c mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app]
[INFO]   build >>> INFO[0506] Using files from context: [/src]
[INFO]   build >>> INFO[0506] ADD . $GOPATH/src/app
[INFO]   build >>> INFO[0506] RUN cd $GOPATH/src/app && go get ./... && go build . && cd /app
[INFO]   build >>> INFO[0506] Checking for cached layer index.docker.io/fabian1991/devspace/cache:c58d7863dc5744a3c34de75247c0fa5f6d0a0bcaeb46981ff1c190470bc277a4...
[INFO]   build >>> INFO[0507] No cached layer found, executing command...
[INFO]   build >>> INFO[0507] cmd: /bin/sh
[INFO]   build >>> INFO[0507] args: [-c cd $GOPATH/src/app && go get ./... && go build . && cd /app]
[INFO]   build >>> INFO[0533] WORKDIR /app
[INFO]   build >>> INFO[0533] cmd: workdir
[INFO]   build >>> INFO[0533] Changed working directory to /app
[INFO]   build >>> INFO[0533] CMD ["$GOPATH/src/app/app"]
[INFO]   build >>> INFO[0533] Taking snapshot of full filesystem...
[INFO]   build >>> ERROR: logging before flag.Parse: E1018 09:42:58.250833      34 metadata.go:142] while reading 'google-dockercfg' metadata: http status code: 404 while fetching url http://metadata.google.internal./computeMetadata/v1/instance/attributes/google-dockercfg
[INFO]   build >>> ERROR: logging before flag.Parse: E1018 09:42:58.255121      34 metadata.go:159] while reading 'google-dockercfg-url' metadata: http status code: 404 while fetching url http://metadata.google.internal./computeMetadata/v1/instance/attributes/google-dockercfg-url
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:bc9ab73e5b14b9fbd3687a4d8c1f1360533d6ee9ffc3f5ecc6630794b40257b7
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:997731689cfbc58c8e74f2a20079338ce66965a40b21f27169b3d5a45ab61cbd
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:1bc310ac474b880a5e4aeec02e6423d1304d137f1a8990074cb3ac6386a0b654
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:e5c3f8c317dc30af45021092a3d76f16ba7aa1ee5f18fec742c84d4960818580
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:193a6306c92af328dbd41bbbd3200a2c90802624cccfe5725223324428110d7f
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:202760eb4a0043cd84cd9971c47052617855ff653abec8ae479e89d369afd500
[INFO]   build >>> 2018/10/18 09:42:59 mounted blob: sha256:a587a86c9dcb9df6584180042becf21e36ecd8b460a761711227b4b06889a005
[INFO]   build >>> 2018/10/18 09:43:00 pushed blob sha256:3299c5b4c55d202d9faab27d32f34cda6622d1f3d2d40c9ff30d16949aed41dc
[INFO]   build >>> 2018/10/18 09:43:03 pushed blob sha256:c425aa94dbf6aa35a519a459df4536d905cc4be08ecb803ad2721a917c27c4d2
[INFO]   build >>> 2018/10/18 09:43:03 index.docker.io/fabian1991/devspace:82i7ApF: digest: sha256:16639a35fd55e1b982e50b6ae1a4039d224c820e5fb4387ba9819d816e922cc1 size: 1578
[DONE] √ Done building image
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/devspace'
[INFO]   Deploying devspace-default with helm
[DONE] √ Deployed helm chart (Release revision: 2)
[DONE] √ Successfully deployed devspace-default
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/kaniko <-> /app (Pod: e388779b2b49465855bb0322057a9fff/devspace-default-864d677f99-t5488)
root@devspace-default-864d677f99-t5488:/go/src/app# 
```

The command created a new kubernetes namespace for you in the devspace-cloud and built your Dockerfile with a kaniko build pod and pushed it to the target docker registry. Afterwards, it deployed the chart in the `chart` folder to that namespace. It also created a new kubectl context for you. You can check the running pods via `kubectl get po`.

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/kaniko` and `/app` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 8080 was forwarded to your local port 8080.  

# Step 2: Start developing

You can start the server now with `go run main.go` in the open terminal. Now navigate in your browser to `localhost:8080` and you should see the output 'Hello World!'.  

Change something in `main.go` locally and re-run `go run main.go`. Now just refresh your browser and you should see the changes immediately.  
