---
title: Command - devspace cleanup images
sidebar_label: images
id: version-v4.0.1-devspace_cleanup_images
original_id: devspace_cleanup_images
---


Deletes all locally created images from docker

## Synopsis

 
```
devspace cleanup images [flags]
```

```
#######################################################
############# devspace cleanup images #################
#######################################################
Deletes all locally created docker images from docker
#######################################################
```
## Options

```
  -h, --help   help for images
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

* [devspace cleanup](/docs/cli/commands/devspace_cleanup)	 - Cleans up resources

