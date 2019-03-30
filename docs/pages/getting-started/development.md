---
title: 3. Develop with DevSpace
---

DevSpace also allows you to work either remotely in a kubernetes cluster or locally with the help of [minikube](https://kubernetes.io/docs/setup/minikube/) and develop applications directly inside kubernetes. The biggest advantages of developing directly inside kubernetes is that your dev environment will be very similar to your production environment and you can have a much greater confidence that everything will work in production when shipping new features.

## Start developing with DevSpace
Developing a DevSpace application is easy, just run this command:
```bash
devspace dev
```

Running `devspace dev` will do the following:
1. Optional: Read the `Dockerfile` and apply in-memory [entrypoint overrides](/docs/cli/development/entrypoint-override)
2. Build a Docker image 
3. Push the Docker image to a container registry
4. Deploy your Helm chart as defined in `chart/`
5. Start [port forwarding](/docs/cli/development/port-forwarding)
6. Start [real-time code synchronization](/docs/cli/development/synchronization)
7. Start [terminal proxy](/docs/cli/development/terminal) (You can also print container logs instead of opening a terminal, see: [terminal](/docs/cli/development/terminal))

> It is highly discouraged to run `devspace dev` multiple times in parallel because the port-forwarding as well as the code synchronization processes will interfere with each other.

## Start your application in the terminal
After running `devspace dev`, you will be directly inside the container terminal, where you can run a command to start your application. Common commands to start your applications in development mode would be:
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```powershell
npm run [start|dev|watch]
```

<!--END_DOCUSAURUS_CODE_TABS-->

By default, `devspace dev` will deploy your containers but your application will not be started, because the entrypoint of your Docker image will be overridden with a `sleep` command. You can also define custom commands for entrypoint overriding. [Learn more about entrypoint overriding.](/docs/cli/development/entrypoint-orverride)

> You can open additional terminals with `devspace enter`. [Learn more about the terminal proxy.](/docs/cli/development/terminal#open-additional-terminals)

## Accessing your application using port-forwarding
After starting your application inside the Space, you can access it via `localhost:[PORT]` because the command `devspace dev` will start port-forwarding for the ports you specified during `devspace init`.

[Learn more about how to configure port forwarding.](/docs/cli/development/port-forwarding)

## Edit your source code and use hot reloading
While the terminal started by `devspace dev` is open, your source code files will be synchronized between your local project and your Space. This allows you to code with your favorite IDE or text editor and use hot reloading tools (e.g. nodemon) to update the application running in your Space in real-time.

## Learn more about developing with DevSpace
Instead of having to run a deployment pipeline to see if everything works correctly, `devspace dev` lets you develop directly inside Kubernetes. This saves a lot of time when building cloud-native software. Developing applications with DevSpace is very powerful and there are many options to define the perfect development workflow for your needs. 

See the following guides to learn more:
- [Use the terminal proxy](/docs/cli/development/synchronization)
- [Configure code synchronization](/docs/cli/development/synchronization)
- [Configure port-forwarding](/docs/cli/development/port-forwarding)
- [Use separate Dockerfile during development](/docs/cli/development/entrypoint-overriding)
- [Override entrypoints](/docs/cli/development/entrypoint-overriding)
- [Monitor and debug applications](/docs/cli/debugging/overview)
- [Learn best practices](/docs/cli/development/best-practices)
