---
title: "Command - devspace create space"
sidebar_label: devspace create space
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


## Flags

```
      --active            Use the new Space as active Space for the current project (default true)
      --cluster string    The cluster to create a space in
  -h, --help              help for space
      --provider string   The cloud provider to use
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

