---
title: "Command - devspace sync"
sidebar_label: devspace sync
---


Starts a bi-directional sync between the target container and the local path

## Synopsis


```
devspace sync [flags]
```

```
#######################################################
################### devspace sync #####################
#######################################################
Starts a bi-directionaly sync between the target container
and the current path:

devspace sync
devspace sync --local-path=subfolder --container-path=/app
devspace sync --exclude=node_modules --exclude=test
devspace sync --pod=my-pod --container=my-container
devspace sync --container-path=/my-path
#######################################################
```


## Flags

```
  -c, --container string           Container name within pod where to execute command
      --container-path string      Container path to use (Default is working directory)
      --download-on-initial-sync   DEPRECATED: Downloads all locally non existing remote files in the beginning (default true)
      --download-only              If set DevSpace will only download files
  -e, --exclude strings            Exclude directory from sync
  -h, --help                       help for sync
      --initial-sync string        The initial sync strategy to use (mirrorLocal, mirrorRemote, preferLocal, preferRemote, preferNewest, keepAll)
  -l, --label-selector string      Comma separated key=value selector list (e.g. release=test)
      --local-path string          Local path to use (Default is current directory
      --no-watch                   Synchronizes local and remote and then stops
      --pick                       Select a pod
      --pod string                 Pod to open a shell to
      --upload-only                If set DevSpace will only upload files
      --verbose                    Shows every file that is synced
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

