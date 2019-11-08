---
title: How To Execute Commands & Open Terminals
sidebar_label: Run Commands & Open Terminals
id: version-v4.2.0-executing-commands
original_id: executing-commands
---

This page shows how to execute commands inside a container and how to start terminal sessions for your containers.

> If you want to enable port-forwarding and/or synchronize files while running a command, [start the terminal using dev mode](#start-terminal-using-dev-mode) or [start the terminal using interactive mode](#start-terminal-using-interactive-mode).

## Run a Single Command
To run a single command inside a container, use:
```bash
devspace enter -- my-command --my-flag=my-value ...
```

## Open a Terminal
To open an interactive terminal session, either use the [localhost UI of DevSpace](../../cli/guides/localhost-ui#start-terminals) or run the following command:
```bash
devspace enter
```

## Start Terminal Using Dev Mode
To start a terminal at the end of `devspace dev` instead of showing the container logs, run:
```bash
devspace dev -t
```

## Start Terminal Using Interactive Mode
To start a terminal at the end of `devspace dev` instead of showing the container logs **and override the `ENTRYPOINT` of an image DevSpace is building**, run:
```bash
devspace dev -i
```
[Learn more about interactive mode.](../../cli/development/configuration/interactive-mode)
