---
title: 3. Develop with DevSpace
id: version-v3.5.18-development
original_id: development
---

DevSpace CLI also allows you to develop applications directly inside a Kubernetes cluster.

The biggest advantages of developing directly inside Kubernetes is that your dev environment will be very similar to your production environment and you can have a much greater confidence that everything will work in production when shipping new features.

> Using DevSpace CLI to develop applications directly inside Kubernetes works with any Kubernetes cluster, i.e. cloud managed, bare-metal, minikube etc.

## Start the development mode
Run the following command to start your application in development mode:
```bash
devspace dev
```

Running `devspace dev` will do the following:
1. Read your application's Dockerfiles and apply in-memory [entrypoint overrides](/docs/development/overrides#configuring-entrypoint-overrides) (optional)
2. Build your application's Dockerfiles as specified in your `devspace.yaml`
3. Push the resulting Docker images to the registries specified in your `devspace.yaml`
4. Deploy your application similar to using `devspace deploy`
5. Start [port forwarding](/docs/development/port-forwarding)
6. Start [real-time code synchronization](/docs/development/synchronization)
7. Start [terminal proxy](/docs/development/terminal) (optional, [see how to configure log streaming instead](/docs/development/terminal#print-logs-instead-of-opening-a-terminal))

> It is highly discouraged to run `devspace dev` multiple times in parallel because multiple instances of port-forwarding and code synchronization will disturb each other. Run `devspace enter` to open additional terminals without port-forwarding and code synchronization.

## Start your application using the terminal proxy
After running `devspace dev`, you will end up inside the terminal of one of your application's containers. In this terminal, you can now run the command to start your application. Common commands to start your applications in development mode would be:
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```powershell
npm run [start|dev|watch]
```

<!--END_DOCUSAURUS_CODE_TABS-->

By default, `devspace dev` will deploy your containers but your application will not be started, because the entrypoints of your Docker images will be overridden with a `sleep` command. You can also define custom commands for entrypoint overriding. 

[Learn more about development overrides.](/docs/development/overrides)

> You can open additional terminals with `devspace enter`. Learn more about how to [use the terminal proxy](/docs/development/terminal#open-additional-terminals).

## Accessing your application using port-forwarding
After starting your application, you can access it via `localhost:[PORT]` because the command `devspace dev` will start port-forwarding for all ports specified in your `devspace.yaml`.

Learn more about how to [configure port forwarding](/docs/development/port-forwarding).

## Edit your source code and use hot reloading
While the terminal that has been started by `devspace dev` is still open, your source code files will be synchronized between your local project and your containers deployed to Kubernetes. This allows you to code with your favorite IDE or text editor and use hot reloading tools (e.g. nodemon) to update the application without having to rebuild your images and redeploy your containers.

## Learn more about developing with DevSpace
Instead of having to run a deployment pipeline to see if everything works correctly, `devspace dev` lets you develop directly inside Kubernetes. This saves a lot of time when building cloud-native software. Developing applications with DevSpace is very powerful and there are many options to define the perfect development workflow for your needs. 

See the following guides to learn more:
- [Use the terminal proxy](/docs/development/terminal)
- [Configure code synchronization](/docs/development/synchronization)
- [Configure port-forwarding](/docs/development/port-forwarding)
- [Use a separate Dockerfile for development](/docs/development/overrides#configuring-a-different-dockerfile-during-devspace-dev)
- [Override entrypoints during development](/docs/development/overrides#configuring-entrypoint-overrides)
- [Example Configurations](https://github.com/devspace-cloud/devspace/tree/master/examples)
