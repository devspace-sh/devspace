---
title: Containerize a project
---

If you want to deploy applications to Kubernetes, you need to package them in Docker images and define appropriate deployment configurations (e.g. Helm charts, Kubernetes manifests). DevSpace can help you containerize your projects and prepare them to be deployed to Kubernetes.

## Containerize a project
The easiest way to containerize an existing project is to run this command:
```bash
devspace init
```

During the initialization process, DevSpace will ask you the following question:
```bash
? Seems like you do not have a Dockerfile. What do you want to do?  [Use arrows to move, type to filter]
> Create a Dockerfile for me                            
  Enter path to your Dockerfile
  Enter path to your Kubernetes manifests
  Enter path to your Helm chart
  Use existing image (e.g. from Docker Hub)
```
If you already have a Dockerfile, let DevSpace use this Dockerfile for packaging your application. If you do not have a Dockerfile yet, you can choose the first option and let DevSpace create a Dockerfile for you. 

In order to create a Dockerfile for your project, DevSpace will ask for the programming language of your application:
```bash
? Select programming language of project  [Use arrows to move, type to filter]
  csharp                       
  go                                     
  java                         
> javascript                               
  none
  php
  python
```

Additionally, DevSpace will ask about the port your application is running on:
```bash
? Which port is the container listening on? (Enter to skip)
```

With the answers to the above questions, DevSpace will not only generate a Dockerfile for you but also add the configuration file `devspace.yaml`. You can freely edit your Dockerfile as well as the DevSpace configuration using any text editor or IDE.

## Containerize a project containing multiple microservices
If you have multiple applications inside a single project directory (i.e. monorepo) and you want to deploy these applications together as a set of microservices, the following procedure is recommended to initialize your project:
1. Make sure each of your services has a Dockerfile. You can run `devspace containerize` in the sub-folder of each of the services to create a Dockerfile. [Learn more below.](#creating-a-dockerfile)
2. Run `devspace init` in the top-level root directory of your project
3. During the init process, choose the option `Enter path to your Dockerfile` and enter the relative path to the Dockerfile of one of your microservices.
4. To add each of the remaining services as deployments, run `devspace add deployment [service-name] --dockerfile="./path/to/your/service/Dockerfile"`

If you already have Helm charts or Kubernetes manifests, you can also add them as a deployment using the following commands:
```bash
devspace add deployment [service-name] --chart="./path/to/your/service/chart"
devspace add deployment [service-name] --manifests="./path/to/your/service/manifests/**"
devspace add deployment [service-name] --image="hub.docker.com/[docker-username]/[my-image]"
```

## Creating a Dockerfile
DevSpace lets you create a Dockerfile for a project without fully initializing your project. In order to generate a Dockerfile for a project run this command inside the root directory of your project:
```bash
devspace containerize
```

DevSpace will ask you which programming language your project uses to create a basic Dockerfile for your project. On command completion, you should see the created Dockerfile inside your project directory. 

### Customizing your Dockerfile

<details>
<summary>
#### Changing the entrypoint (start script) in your Dockerfile
</summary>

A common issue why a container cannot be executed and this problem is usually discovered only later is because of a wrongly defined entrypoint that is not existent. In the nodejs Dockerfile example:

```Dockerfile
FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

# This is the command that will be executed 
CMD ["npm", "start"]
```

the line `CMD ["npm", "start"]` specifies the executed command on container start. If your `package.json` has no start script defined, the container will fail to execute. Let's say you want to change the start command to `node index.js`, you would rewrite the Dockerfile like this:

```Dockerfile
FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

# This is the command that will be executed 
CMD ["node", "index.js"]
```

</details>


### Troubleshooting your Dockerfile
In order to verify that your Dockerfile is working corretly, you can simply build it with this command:
```bash
docker build .
```
*Running the above command requires Docker to be installed on your computer.*

> In general it is a good idea to look at the official [Docker documentation](https://docs.docker.com/develop/) and the [best practices how to write a Dockerfile](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/).

#### Common issues with your Dockerfile

<details>
<summary>
##### Error: No such file or directory
</summary>

A common issue why a docker build with the predefined docker files is because some files are missing in the project we assume you habe. Take a look a the nodejs example Dockerfile:
```Dockerfile
FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

CMD ["npm", "start"]
```

In this Dockerfile it is assumed your project has a `package.json`, however not all nodejs projects have such a file. Running `docker build .` without a `package.json` can yield the following result: 

```Bash
Sending build context to Docker daemon  283.6kB
Step 1/10 : FROM node:8.11.4
 ---> 8198006b2b57
[...]
Step 4/10 : COPY package.json .
COPY failed: stat /var/lib/docker/tmp/docker-builder846046622/package.json: no such file or directory
```

This error indicates that docker cannot find the file `package.json` in your project, but it is required in Step 4, hence docker build fails. You can adjust the Dockerfile like this:

```
FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY . .

CMD ["node", "index.js"]
```

and skip the dependency installation. Bear in mind that you also have to change the entrypoint of the container, since `npm start` will not work without a `package.json`.
</details>
