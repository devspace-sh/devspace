---
title: "Command - devspace remove sync"
sidebar_label: devspace remove sync
---


Remove sync paths from the devspace

## Synopsis


```
devspace remove sync [flags]
```

```
#######################################################
############### devspace remove sync ##################
#######################################################
Remove sync paths from the devspace

How to use:
devspace remove sync --local=app
devspace remove sync --container=/app
devspace remove sync --label-selector=release=test
devspace remove sync --all
#######################################################
```


## Flags

```
      --all                     Remove all configured sync paths
      --container string        Absolute container path to remove
  -h, --help                    help for sync
      --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --local string            Relative local path to remove
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

