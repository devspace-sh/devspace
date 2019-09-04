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
1. Read your application's Dockerfile(s) and apply in-memory [entrypoint overrides](/docs/cli/development/configuration/dev-overrides#configuring-entrypoint-overrides) (optional)
2. Build your application's Dockerfile(s) as specified in your `devspace.yaml`
3. Push the resulting Docker images to the registries specified in your `devspace.yaml`
4. Deploy your application similar to using `devspace deploy`
5. Start [port forwarding](/docs/cli/development/configuration/port-forwarding)
6. Start [real-time code synchronization](/docs/cli/development/configuration/file-synchronization)
7. Either [streams the logs of your containers](/docs/cli/development/configuration/terminal-proxy-logs#print-logs-instead-of-opening-a-terminal) OR starts the [terminal proxy](/docs/cli/development/configuration/terminal-proxy-logs)


<img src="/img/processes/development-process-devspace.svg" alt="DevSpace Development Process" style="width: 100%;">

> It is highly discouraged to run `devspace dev` multiple times in parallel because multiple instances of port-forwarding and code synchronization will disturb each other. Run `devspace enter` to open a simple terminal session without port-forwarding and code synchronization.

## Access Your Application via Port-Forwarding
After starting your application, you can access it via `localhost:[PORT]` because the command `devspace dev` will start port-forwarding for all ports specified in your `devspace.yaml`. 

> If you want additional ports to be fowarded, you can add them to `devspace.yaml` or simply run `devspace add port [port]`.

Learn more about how to [configure port forwarding](/docs/cli/development/configuration/port-forwarding).

## Edit your source code and use hot reloading
While `devspace dev` is still running, your source code files will be synchronized between your local project and your containers running inside Kubernetes. This allows you to code with your favorite IDE or text editor and use hot reloading tools (e.g. nodemon) to update the application without having to rebuild your images and redeploy your containers.

> This step requires your application to start with a hot reloading tool, e.g. nodemon. You have three options to change how your container will be started:
> 1. Simply edit the ENTRYPOINT in your Dockerfile (easy but not always an option)
> 2. Start the development in interactive mode using `devspace dev -i` and run the start command manually after the terminal opens (e.g. `npm run develop`)
> 3. [Tell DevSpace to override the ENTRYPOINT when using the development mode](/docs/cli/development/configuration/dev-overrides#configuring-entrypoint-overrides)

## Learn more about developing with DevSpace
Instead of having to run a deployment pipeline to see if everything works correctly, `devspace dev` lets you develop directly inside Kubernetes. This saves a lot of time when building cloud-native software. Developing applications with DevSpace is very powerful and there are many options to define the perfect development workflow for your needs. 

See the following guides to learn more:
- [Example Configurations](https://github.com/devspace-cloud/devspace#configuration-examples)
- [Configure Dependencies](/docs/cli/deployment/advanced/dependencies) (microservices from other repositories)
