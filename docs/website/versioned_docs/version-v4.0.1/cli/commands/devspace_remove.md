---
title: Command - devspace remove
sidebar_label: devspace remove
id: version-v4.0.1-devspace_remove
original_id: devspace_remove
---


Changes devspace configuration

## Synopsis


```
#######################################################
################## devspace remove ####################
#######################################################
```
## Options

```
  -h, --help   help for remove
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
* [devspace remove cluster](../../cli/commands/devspace_remove_cluster)	 - Removes a connected cluster
* [devspace remove context](../../cli/commands/devspace_remove_context)	 - Removes a kubectl-context
* [devspace remove deployment](../../cli/commands/devspace_remove_deployment)	 - Removes one or all deployments from devspace configuration
* [devspace remove image](../../cli/commands/devspace_remove_image)	 - Removes one or all images from the devspace
* [devspace remove port](../../cli/commands/devspace_remove_port)	 - Removes forwarded ports from a devspace
* [devspace remove provider](../../cli/commands/devspace_remove_provider)	 - Removes a cloud provider from the configuration
* [devspace remove space](../../cli/commands/devspace_remove_space)	 - Removes a cloud space
* [devspace remove sync](../../cli/commands/devspace_remove_sync)	 - Remove sync paths from the devspace
