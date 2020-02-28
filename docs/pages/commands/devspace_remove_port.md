---
title: "Command - devspace remove port"
sidebar_label: devspace remove port
---


Removes forwarded ports from a devspace

## Synopsis


```
devspace remove port [flags]
```

```
#######################################################
############### devspace remove port ##################
#######################################################
Removes port mappings from the devspace configuration:
devspace remove port 8080,3000
devspace remove port --label-selector=release=test
devspace remove port --all
#######################################################
```


## Flags

```
      --all                     Remove all configured ports
  -h, --help                    help for port
      --label-selector string   Comma separated key=value selector list (e.g. release=test)
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

