---
title: 4. Develop with DevSpace
---

DevSpace also allows you to connect your local project to a Space and develop applications directly inside this Space. The biggest advantages of developing directly inside a Space is that your dev environment will be very similar to your production environment and you can have a much greater confidence that everything will work in production when shipping new features.

## Create a Space for Development
Create a new Space called `development` with the following command:
```bash
devspace create space development
```
Now, you should have 2 Spaces called `production` and `development`. To get a list of all your Spaces, run:
```bash
devspace list spaces
```
You should see an output similar to this one:
```bash
TODO
```

> You should use separate Spaces for development, staging and production. Additionally, everyone working on a project together should use a separate Space for development. [Learn more best practices.](../development/best-practices)

## Select a Space for development
To be sure that you are connected with the correct Space, you should always actively switch to your development Space. In this case, we want to develop within the Space called `development` that we created above. So, we run this command:
```bash
devspace use space development
```

## Start developing with DevSpace
Now, you can start to develop within your Space with this command:
```bash
devspace dev
```

> It is highly discouraged to run `devspace dev` in production Spaces. [Learn more best practices.](../development/best-practices)

Running `devspace dev` will do the following:
1. Read the `Dockerfile` and apply in-memory [entrypoint overrides](../development/entrypoint-override) (optional)
2. Build a Docker image using the (overridden) `Dockerfile`
3. Push this Docker image to the [DevSpace Container Registry (dscr.io)](../images/internal-registry)
4. Deploy your Helm chart as defined in `chart/`
5. Start [port forwarding](../development/port-forwarding)
6. Start [real-time code synchronization](../development/synchronization)
7. Start [terminal proxy](../development/terminal)

> It is highly discouraged to run `devspace dev` multiple times in parallel because the port-forwarding as well as the code synchronization processes will interfere with each other. [Learn more best practices.](../development/best-practices)

## Start your application within the Space
After running `devspace dev`, you will be directly inside the container terminal, where you can run a command to start your application. Common commands to start your applications in development mode would be:
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```powershell
npm run [start|dev|watch]
```

<!--END_DOCUSAURUS_CODE_TABS-->

By default, `devspace dev` will deploy your containers but your application will not be started, because the entrypoint of your Docker image will be overridden with a `sleep` command. You can also define custom commands for entrypoint overriding. [Learn more about entrypoint overriding.](../development/entrypoint-orverride)

> You can open additional terminals with `devspace enter`. [Learn more about the terminal proxy.](../development/terminal#open-additional-terminals)

## Accessing your application using port-forwarding
After starting your application inside the Space, you can access it via `localhost:[PORT]` because the command `devspace dev` will start port-forwarding for the ports you specified during `devspace init`.

[Learn more about how to configure port forwarding.](../development/port-forwarding)

> Instead of using port-forwarding, you could also access your application using the automatically generated URL `my-space-url.devspace.host`. [Learn more about Space URLs.](../space/what-are-spaces#auto-generated-urls)

## Edit your source code and use hot reloading
While the terminal started by `devspace dev` is open, your source code files will be synchronized between your local project and your Space. This allows you to code with your favorite IDE or text editor and use hot reloading tools (e.g. nodemon) to update the application running in your Space in real-time.

## Learn more about developing with DevSpace
Instead of having to run a deployment pipeline to see if everything works correctly, `devspace dev` lets you develop directly inside Kubernetes. This saves a lot of time when building cloud-native software. Developing applications with DevSpace is very powerful and there are many options to define the perfect development workflow for your needs. 

See the following guides to learn more:
- [Use the terminal proxy](../development/synchronization)
- [Configure code synchronization](../development/synchronization)
- [Configure port-forwarding](../development/port-forwarding)
- [Override entrypoints](../development/entrypoint-overriding)
- [Monitor and debug applications](../development/debugging)
- [Learn best practices](../development/best-practices)
