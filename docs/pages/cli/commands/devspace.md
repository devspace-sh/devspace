---
title: "Command - devspace"
sidebar_label: devspace
---


Welcome to the DevSpace!

## Synopsis

```
DevSpace accelerates developing, deploying and debugging applications with Docker and Kubernetes. Get started by running the init command in one of your projects:

	devspace init
```
## Options

```
      --debug                 Prints the stack trace if an error occurs
  -h, --help                  help for devspace
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

## See Also

* [devspace add](../../cli/commands/devspace_add)	 - Change the DevSpace configuration
* [devspace analyze](../../cli/commands/devspace_analyze)	 - Analyzes a kubernetes namespace and checks for potential problems
* [devspace attach](../../cli/commands/devspace_attach)	 - Attaches to a container
* [devspace build](../../cli/commands/devspace_build)	 - Builds all defined images and pushes them
* [devspace cleanup](../../cli/commands/devspace_cleanup)	 - Cleans up resources
* [devspace connect](../../cli/commands/devspace_connect)	 - Connect an external cluster to devspace cloud
* [devspace create](../../cli/commands/devspace_create)	 - Create spaces in the cloud
* [devspace deploy](../../cli/commands/devspace_deploy)	 - Deploy the project
* [devspace dev](../../cli/commands/devspace_dev)	 - Starts the development mode
* [devspace enter](../../cli/commands/devspace_enter)	 - Open a shell to a container
* [devspace init](../../cli/commands/devspace_init)	 - Initializes DevSpace in the current folder
* [devspace list](../../cli/commands/devspace_list)	 - Lists configuration
* [devspace login](../../cli/commands/devspace_login)	 - Log into DevSpace Cloud
* [devspace logs](../../cli/commands/devspace_logs)	 - Prints the logs of a pod and attaches to it
* [devspace open](../../cli/commands/devspace_open)	 - Opens the space in the browser
* [devspace purge](../../cli/commands/devspace_purge)	 - Delete deployed resources
* [devspace remove](../../cli/commands/devspace_remove)	 - Changes devspace configuration
* [devspace reset](../../cli/commands/devspace_reset)	 - Resets an cluster token
* [devspace run](../../cli/commands/devspace_run)	 - Run executes a predefined command
* [devspace set](../../cli/commands/devspace_set)	 - Make global configuration changes
* [devspace status](../../cli/commands/devspace_status)	 - Show the current status
* [devspace sync](../../cli/commands/devspace_sync)	 - Starts a bi-directional sync between the target container and the local path
* [devspace ui](../../cli/commands/devspace_ui)	 - Opens the management ui in the browser
* [devspace update](../../cli/commands/devspace_update)	 - Updates the current config
* [devspace upgrade](../../cli/commands/devspace_upgrade)	 - Upgrade the DevSpace CLI to the newest version
* [devspace use](../../cli/commands/devspace_use)	 - Use specific config
