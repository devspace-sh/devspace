---
title: "Command - devspace login"
sidebar_label: devspace login
---


Log into DevSpace Cloud

## Synopsis


```
devspace login [flags]
```

```
#######################################################
################### devspace login ####################
#######################################################
If no --key is supplied the browser will be opened 
and the login page is shown

Example:
devspace login
devspace login --key myaccesskey
#######################################################
```


## Flags

```
  -h, --help              help for login
      --key string        Access key to use
      --provider string   Provider to use
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

