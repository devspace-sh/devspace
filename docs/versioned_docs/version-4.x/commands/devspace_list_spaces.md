---
title: "Command - devspace list spaces"
sidebar_label: devspace list spaces
---


Lists all user spaces

## Synopsis


```
devspace list spaces [flags]
```

```
#######################################################
############### devspace list spaces ##################
#######################################################
List all user cloud spaces

Example:
devspace list spaces
devspace list spaces --cluster my-cluster
devspace list spaces --all
#######################################################
```


## Flags

```
      --all               List all spaces the user has access to in all clusters (not only created by the user)
      --cluster string    List all spaces in a certain cluster
  -h, --help              help for spaces
      --name string       Space name to show (default: all)
      --provider string   Cloud Provider to use
```


## Global & Inherited Flags

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

