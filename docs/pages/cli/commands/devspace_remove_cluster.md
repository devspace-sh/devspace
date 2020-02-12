---
title: "Command - devspace remove cluster"
sidebar_label: cluster
---


Removes a connected cluster

## Synopsis


```
devspace remove cluster [flags]
```

```
#######################################################
############# devspace remove cluster #################
#######################################################
Removes a connected cluster 

Example:
devspace remove cluster my-cluster
#######################################################
```
## Options

```
  -h, --help              help for cluster
      --provider string   The cloud provider to use
  -y, --yes               Ignores all questions and deletes the cluster with all services and spaces
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

* [devspace remove](../../cli/commands/devspace_remove)	 - Changes devspace configuration
