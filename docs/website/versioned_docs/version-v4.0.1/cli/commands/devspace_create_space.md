---
title: Command - devspace create space
sidebar_label: space
id: version-v4.0.1-devspace_create_space
original_id: devspace_create_space
---


Create a new cloud space

## Synopsis


```
devspace create space [flags]
```

```
#######################################################
############### devspace create space #################
#######################################################
Creates a new space

Example:
devspace create space myspace
#######################################################
```
## Options

```
      --active            Use the new Space as active Space for the current project (default true)
      --cluster string    The cluster to create a space in
  -h, --help              help for space
      --provider string   The cloud provider to use
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

* [devspace create](/docs/cli/commands/devspace_create)	 - Create spaces in the cloud

