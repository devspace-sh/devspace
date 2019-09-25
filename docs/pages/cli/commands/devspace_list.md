---
title: "Command - devspace list"
sidebar_label: devspace list
---


Lists configuration

## Synopsis


```
#######################################################
#################### devspace list ####################
#######################################################
```
## Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --debug                 Prints the stack trace if an error occurs
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

## See Also
* [devspace list available-components](../../cli/commands/devspace_list_available-components)	 - Lists all available components
* [devspace list clusters](../../cli/commands/devspace_list_clusters)	 - Lists all connected clusters
* [devspace list commands](../../cli/commands/devspace_list_commands)	 - Lists all custom DevSpace commands
* [devspace list deployments](../../cli/commands/devspace_list_deployments)	 - Lists and shows the status of all deployments
* [devspace list ports](../../cli/commands/devspace_list_ports)	 - Lists port forwarding configurations
* [devspace list profiles](../../cli/commands/devspace_list_profiles)	 - Lists all DevSpace profiles
* [devspace list providers](../../cli/commands/devspace_list_providers)	 - Lists all providers
* [devspace list spaces](../../cli/commands/devspace_list_spaces)	 - Lists all user spaces
* [devspace list sync](../../cli/commands/devspace_list_sync)	 - Lists sync configuration
* [devspace list vars](../../cli/commands/devspace_list_vars)	 - Lists the vars in the active config
