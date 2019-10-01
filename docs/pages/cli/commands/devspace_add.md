---
title: "Command - devspace add"
sidebar_label: devspace add
---


Change the DevSpace configuration

## Synopsis


```
#######################################################
#################### devspace add #####################
#######################################################
```
## Options

```
  -h, --help   help for add
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
* [devspace add deployment](../../cli/commands/devspace_add_deployment)	 - Add a deployment
* [devspace add image](../../cli/commands/devspace_add_image)	 - Add an image
* [devspace add port](../../cli/commands/devspace_add_port)	 - Add a new port forward configuration
* [devspace add provider](../../cli/commands/devspace_add_provider)	 - Adds a new cloud provider to the configuration
* [devspace add sync](../../cli/commands/devspace_add_sync)	 - Add a sync path
