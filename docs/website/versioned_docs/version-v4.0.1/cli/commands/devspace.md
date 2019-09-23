---
title: Command - devspace
sidebar_label: devspace
id: version-v4.0.1-devspace
original_id: devspace
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

* [devspace add](/docs/cli/commands/devspace_add)	 - Change the DevSpace configuration
* [devspace analyze](/docs/cli/commands/devspace_analyze)	 - Analyzes a kubernetes namespace and checks for potential problems
* [devspace attach](/docs/cli/commands/devspace_attach)	 - Attaches to a container
* [devspace build](/docs/cli/commands/devspace_build)	 - Builds all defined images and pushes them
* [devspace cleanup](/docs/cli/commands/devspace_cleanup)	 - Cleans up resources
* [devspace connect](/docs/cli/commands/devspace_connect)	 - Connect an external cluster to devspace cloud
* [devspace create](/docs/cli/commands/devspace_create)	 - Create spaces in the cloud
* [devspace deploy](/docs/cli/commands/devspace_deploy)	 - Deploy the project
* [devspace dev](/docs/cli/commands/devspace_dev)	 - Starts the development mode
* [devspace enter](/docs/cli/commands/devspace_enter)	 - Open a shell to a container
* [devspace init](/docs/cli/commands/devspace_init)	 - Initializes DevSpace in the current folder
* [devspace list](/docs/cli/commands/devspace_list)	 - Lists configuration
* [devspace login](/docs/cli/commands/devspace_login)	 - Log into DevSpace Cloud
* [devspace logs](/docs/cli/commands/devspace_logs)	 - Prints the logs of a pod and attaches to it
* [devspace open](/docs/cli/commands/devspace_open)	 - Opens the space in the browser
* [devspace purge](/docs/cli/commands/devspace_purge)	 - Delete deployed resources
* [devspace remove](/docs/cli/commands/devspace_remove)	 - Changes devspace configuration
* [devspace reset](/docs/cli/commands/devspace_reset)	 - Resets an cluster token
* [devspace run](/docs/cli/commands/devspace_run)	 - Run executes a predefined command
* [devspace set](/docs/cli/commands/devspace_set)	 - Make global configuration changes
* [devspace status](/docs/cli/commands/devspace_status)	 - Show the current status
* [devspace sync](/docs/cli/commands/devspace_sync)	 - Starts a bi-directional sync between the target container and the local path
* [devspace ui](/docs/cli/commands/devspace_ui)	 - Opens the management ui in the browser
* [devspace update](/docs/cli/commands/devspace_update)	 - Updates the current config
* [devspace upgrade](/docs/cli/commands/devspace_upgrade)	 - Upgrade the DevSpace CLI to the newest version
* [devspace use](/docs/cli/commands/devspace_use)	 - Use specific config

