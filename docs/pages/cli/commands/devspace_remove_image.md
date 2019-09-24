---
title: "Command - devspace remove image"
sidebar_label: image
---


Removes one or all images from the devspace

## Synopsis


```
devspace remove image [flags]
```

```
#######################################################
############ devspace remove image ####################
#######################################################
Removes one or all images from a devspace:
devspace remove image default
devspace remove image --all
#######################################################
```
## Options

```
      --all    Remove all images
  -h, --help   help for image
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

* [devspace remove](/docs/cli/commands/devspace_remove)	 - Changes devspace configuration

