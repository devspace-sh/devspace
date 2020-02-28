---
title: "Command - devspace remove"
sidebar_label: devspace remove
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
* [devspace remove cluster](devspace_remove_cluster.md)	 - Removes a connected cluster
* [devspace remove context](devspace_remove_context.md)	 - Removes a kubectl-context
* [devspace remove deployment](devspace_remove_deployment.md)	 - Removes one or all deployments from devspace configuration
* [devspace remove image](devspace_remove_image.md)	 - Removes one or all images from the devspace
* [devspace remove port](devspace_remove_port.md)	 - Removes forwarded ports from a devspace
* [devspace remove provider](devspace_remove_provider.md)	 - Removes a cloud provider from the configuration
* [devspace remove space](devspace_remove_space.md)	 - Removes a cloud space
* [devspace remove sync](devspace_remove_sync.md)	 - Remove sync paths from the devspace
