---
title: "Command - devspace reset key"
sidebar_label: devspace reset key
---


Resets a cluster key

## Synopsis


```
devspace reset key [flags]
```

```
#######################################################
############### devspace reset key ####################
#######################################################
Resets a key for a given cluster. Useful if the key 
cannot be obtained anymore. Needs cluster access scope

Examples:
devspace reset key my-cluster
#######################################################
```


## Flags

```
  -h, --help              help for key
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

