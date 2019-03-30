---
title: Containerize a project
---

DevSpace requires your project to have at least one Dockerfile that can be built with docker. If you don't have a working Dockerfile, DevSpace can help you create one.

# Containerize your existing application

DevSpace can create a default Dockerfile for your project based on the primary programming language it has detected. In order to generate a Dockerfile for your project run this command in your shell at the project root:
```bash
devspace containerize
```

DevSpace will ask you which programming language your project uses and based on the answer a predefined Dockerfile template will be created for your project. On command completion, you should see the created Dockerfile at your project root. In order to verify that the Dockerfile is working, you can run the following command:
```bash
docker build .
```

If you see an output similar to this, everything is working correctly:
```bash
$ docker build .
Sending build context to Docker daemon  283.6kB
Step 1/9 : FROM node:8.11.4
 ---> 8198006b2b57
[...]
Step 9/9 : CMD ["npm", "start"]
 ---> 277ddaad567a
Successfully built 277ddaad567a
```

# Troubleshooting

In general it is a good idea to look at the official [docker documentation](https://docs.docker.com/develop/) and the [best practices how to write a Dockerfile](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/). Some other common issues to look out for are listed below:

<details>
<summary>
### application cannot be accessed with space url
</summary>

This could be caused by several problems. Run `devspace analyze` and check if there are any issues during container startup. If not [routing](/docs/cloud/spaces/configure-networking) to your container could be an issue. Make sure your container is listening on `0.0.0.0` and not `localhost` and on the same port as specified in `chart/values.yaml` under components.service.containerPort.

</details>

<details>
<summary>
### No such file or directory
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
<details>
<summary>
### Entrypoint
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
