---
title: Configuring Terminal & Log Streaming
sidebar_label: Terminal & Logs
---

After running `devspace dev`, DevSpace CLI will open a terminal proxy, which lets you run commands within your deployed containers.

## Start your application via terminal proxy
Starting your application with the terminal proxy works the same way as starting it with your local terminal. Common commands to start your applications in development mode would be:
<!--DOCUSAURUS_CODE_TABS-->
<!--Node.js-->
```powershell
npm run [start|dev|watch]
```

<!--END_DOCUSAURUS_CODE_TABS-->

By default, `devspace dev` will deploy your containers but your application will not be started, because the entrypoint of your Docker image will be overridden with a `sleep` command. You can also define custom commands for overriding entrypoints. [Learn more about entrypoint overriding.](/docs/development/overrides#configuring-entrypoint-overrides)

## Print logs instead of opening a terminal 

If you rather want to print the container logs instead of opening a terminal to the container you can define this in the `devspace.yaml`:

```yaml
dev:
  selectors:
  - name: default
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
  terminal:
    selector: default
    containerName: nodejs
    # Next line tells devspace to show logs instead of terminal
    disabled: true
```

## Open additional terminals
You can open additional terminals, simply run the following command:
```bash
devspace enter
```

> **Do not run `devspace dev` to open additional terminals.**  
> It is highly discouraged to run `devspace dev` multiple times in parallel because the port-forwarding as well as the code synchronization processes will interfere with each other.

Use the command `devspace enter [COMMAND]` to run a command direclty after opening the terminal.
```bash
devspace enter bash
devspace enter npm start
```
The first command listed above would open an interactive terminal proxy and start a bash terminal within the existing terminal session. The second command would open an interactive terminal proxy and then run the command `npm start`.

## Open terminals for other containers
The `devspace enter` command supports a variety of flags to open terminals for containers other than the detault container specified in the configuration.
```bash
devspace enter -p                   # --pick | Show a list of pods and containers to enter into
devspace enter -c my-container      # --container | Select container "my-container" within the default terminal component
devspace enter -s mysql             # --selector | Use the selector with name "mysql" to start the terminal proxy
devspace enter -l "release=test"    # --label-sector | Use the label selector "release=test" to start the terminal proxy
```
[See the full specification for `devspace enter`.](/docs/cli-commands/enter)

## Configure the terminal proxy
The configuration for the terminal proxy can be set within the `dev.terminal` section of `devspace.yaml`.
```yaml
dev:
  selectors:
  - name: default
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
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
The defaults for opening the terminal proxy can be configured in the `dev.terminal` section within `../devspace.yaml`.
</details>
