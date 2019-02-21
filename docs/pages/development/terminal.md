---
title: Use the terminal proxy
---

After running `devspace dev`, DevSpace.cli will open a terminal proxy, which lets you run commands within your deployed containers.

## Start your application via terminal proxy
Starting your application with the terminal proxy works the same way as starting it with your local terminal. Common commands to start your applications in development mode would be:
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```powershell
npm run [start|dev|watch]
```

<!--END_DOCUSAURUS_CODE_TABS-->

By default, `devspace dev` will deploy your containers but your application will not be started, because the entrypoint of your Docker image will be overwritten with a `sleep` command. You can also define custom commands for entrypoint overwriting. [Learn more about entrypoint overwrites.](../development/entrypoint-overwrites)

## Open additional terminals
You can open additional terminals, simply run the following command:
```bash
devspace enter
```

> **Do not run `devspace dev` to open additional terminals.**  
> It is highly discouraged to run `devspace dev` multiple times in parallel because the port-forwarding as well as the code synchronization processes will interfere with each other. [Learn more best practices.](./best-practices)

Use the command `devspace enter [COMMAND]` to run a command direclty after opening the terminal.
```bash
devspace enter bash
devspace enter npm start
```
The first command listed above would open an interactive terminal proxy and start a bash terminal within the existing terminal session. The second command would open an interactive terminal proxy and then run the command `npm start`.

## Open terminals for other containers
The `devspace enter` command supports a variety of flags to open terminals for containers other than the detault container specified in the configuration.
```bash
devspace enter -c my-container  # Select container "my-container" within the default terminal component
devspace enter -s mysql         # Use the selector with name "mysql" to start the terminal proxy
devspace enter -l release=test  # Use the label selector "release=test" to start the terminal proxy
```
[See the full specification for `devspace enter`.](../cli/enter)

## Configure the terminal proxy
The configuration for the terminal proxy can be set within the `dev.terminal` section of `.devspace/config.yaml`.
```yaml
dev:
  terminal:
    selector: default
    containerName: nodejs
```
The above example defines that by default the terminal proxy should be opened for the container with name `nodejs` within the pod that will be selected through the selector with the name `default`.

> If `containerName` is not specified, the terminal proxy will be opened for the first container within the pod that has been selected with the given `selector`.

---
## FAQ

<details>
<summary>
### Where are the defaults for opening the terminal proxy defined?
</summary>
The defaults for opening the terminal proxy can be configured in the `dev.terminal` section within `.devspace/config.yaml`.
</details>
