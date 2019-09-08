---
title: 3. Develop inside Kubernetes
---

DevSpace also allows you to develop applications directly inside a Kubernetes cluster.

The biggest advantages of developing directly inside Kubernetes is that your dev environment will be very similar to your production environment and you can have a much greater confidence that everything will work in production when shipping new features.

## Start Development
Run the following command to start your application in development mode:
```bash
devspace dev
```

Running `devspace dev` will do the following:
1. Build your Docker images similar to using `devspace build` or `devspace deploy`
2. Deploy your application similar to using `devspace deploy`
3. Start [port-forwarding](/docs/cli/development/configuration/port-forwarding)
4. Either [streams the logs of your containers](/docs/cli/development/configuration/terminal-proxy-logs#print-logs-instead-of-opening-a-terminal) OR starts the [terminal proxy](/docs/cli/development/configuration/terminal-proxy-logs)
5. Start the [real-time code synchronization](/docs/cli/development/configuration/file-synchronization), so you can reload your application without having to redeploy or restart your containers (using hot reloading directly within your containers)


<img src="/img/processes/development-process-devspace.svg" alt="DevSpace Development Process" style="width: 100%;">

> It is highly discouraged to run `devspace dev` multiple times in parallel because multiple instances of port-forwarding and file synchronization will disturb each other. Instead:
> - Run `devspace enter` to open a terminal session without port-forwarding and file synchronization
> - Run `devspace logs` to open log streaming without port-forwarding and file synchronization

## Access Your Application via Port-Forwarding
After starting your application, you can access it via `localhost:[PORT]` because the command `devspace dev` will start port-forwarding for all ports specified in the `dev.ports` section of your project's `devspace.yaml`. 

> If you want additional ports to be fowarded, you can add them manually or simply run `devspace add port [port]`.

Learn more about how to [configure port forwarding](/docs/cli/development/configuration/port-forwarding).

## Code & Reload Your Application
While `devspace dev` is still running, your source code files will be synchronized between your local project and your containers running inside Kubernetes. This allows you to code with your favorite IDE or text editor and use hot reloading tools (e.g. `nodemon`) to update the application without having to rebuild your images or redeploy your containers.

> This step requires your application to start with a hot reloading tool, e.g. nodemon. To do this, you have two options:
> - **Option 1: Edit the ENTRYPOINT in your Dockerfile** (easy and simple to share with others but often requires to setup a [separate profile for staging or production-like deployments](#TODO))
> - **Option 2: Start the development in interactive mode** using `devspace dev -i` and run the start command manually after the terminal opens, e.g. `npm run develop` (quick and non-intrusive but hard to share with your team mates)
> If you are using one of the quickstart projects, you can see that the ENTRYPOINT in your Dockerfile is already starting the application using hot reloading (Option 1).

**Now that you started your application using hot reloading, you can edit a file, hit save and see how DevSpace uploads it to your containers which triggers your application to reload.**

## Learn more about developing with DevSpace
Instead of having to run a deployment pipeline to see if everything works correctly, `devspace dev` lets you develop directly inside Kubernetes. This saves a lot of time when building cloud-native software. in-cluster development with DevSpace is very powerful and there are many options to define the perfect development workflow for your needs. 

See the following links for:
- [Example Configurations](https://github.com/devspace-cloud/devspace#configuration-examples) for common use cases
- [Example Projects](https://github.com/devspace-cloud/devspace/tree/master/examples) with fully-fledged configuration files
- [Image Building - Workflow & Basics](/docs/cli/image-building/workflow-basics)
- [Deployment - Workflow & Basics](/docs/cli/deployment/workflow-basics)
- [Development - Workflow & Basics](/docs/cli/development/workflow-basics)
