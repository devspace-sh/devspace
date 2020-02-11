---
title: Command - devspace cleanup
sidebar_label: devspace cleanup
id: version-v4.0.1-devspace_cleanup
original_id: devspace_cleanup
---


Cleans up resources

## Synopsis


```
#######################################################
################## devspace cleanup ###################
#######################################################
```
## Options

```
  -h, --help   help for cleanup
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
* [devspace cleanup images](../../cli/commands/devspace_cleanup_images)	 - Deletes all locally created images from docker
