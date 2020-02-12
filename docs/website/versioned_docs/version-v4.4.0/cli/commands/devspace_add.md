---
title: Command - devspace add
sidebar_label: devspace add
id: version-v4.4.0-devspace_add
original_id: devspace_add
---


Convenience command: adds something to devspace.yaml

## Synopsis


```
#######################################################
#################### devspace add #####################
#######################################################
Adds config sections to devspace.yaml
```
## Options

```
  -h, --help   help for add
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
* [devspace add deployment](../../cli/commands/devspace_add_deployment)	 - Adds a deployment to devspace.yaml
* [devspace add image](../../cli/commands/devspace_add_image)	 - Add an image
* [devspace add port](../../cli/commands/devspace_add_port)	 - Add a new port forward configuration
* [devspace add provider](../../cli/commands/devspace_add_provider)	 - Adds a new cloud provider to the configuration
* [devspace add sync](../../cli/commands/devspace_add_sync)	 - Add a sync path
