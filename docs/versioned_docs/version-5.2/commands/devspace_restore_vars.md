---
title: "Command - devspace restore vars"
sidebar_label: devspace restore vars
---


Restores variable values from kubernetes

## Synopsis


```
devspace restore vars [flags]
```

```
#######################################################
############### devspace restore vars #################
#######################################################
Restores devspace config variable values from a kubernetes
secret. 

Examples:
devspace restore vars
devspace restore vars --namespace test 
devspace restore vars --vars-secret my-secret
#######################################################
```


## Flags

```
  -h, --help                 help for vars
      --vars-secret string   The secret to restore the variables from (default "devspace-vars")
```


## Global & Inherited Flags

```
      --config string         The devspace config file to use
      --debug                 Prints the stack trace if an error occurs
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --profile-refresh       If true will pull and re-download profile parent sources
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

