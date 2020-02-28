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
      --config string         The devspace config file to use
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
* [devspace list available-components](devspace_list_available-components.md)	 - Lists all available components
* [devspace list clusters](devspace_list_clusters.md)	 - Lists all connected clusters
* [devspace list commands](devspace_list_commands.md)	 - Lists all custom DevSpace commands
* [devspace list contexts](devspace_list_contexts.md)	 - Lists all kube contexts
* [devspace list deployments](devspace_list_deployments.md)	 - Lists and shows the status of all deployments
* [devspace list namespaces](devspace_list_namespaces.md)	 - Lists all namespaces in the current context
* [devspace list ports](devspace_list_ports.md)	 - Lists port forwarding configurations
* [devspace list profiles](devspace_list_profiles.md)	 - Lists all DevSpace profiles
* [devspace list providers](devspace_list_providers.md)	 - Lists all providers
* [devspace list spaces](devspace_list_spaces.md)	 - Lists all user spaces
* [devspace list sync](devspace_list_sync.md)	 - Lists sync configuration
* [devspace list vars](devspace_list_vars.md)	 - Lists the vars in the active config
