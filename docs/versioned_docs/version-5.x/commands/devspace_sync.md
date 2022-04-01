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
  -c, --container string           Container name within pod where to sync to
      --container-path string      Container path to use (Default is working directory)
      --download-on-initial-sync   DEPRECATED: Downloads all locally non existing remote files in the beginning (default true)
      --download-only              If set DevSpace will only download files
  -e, --exclude strings            Exclude directory from sync
  -h, --help                       help for sync
      --initial-sync string        The initial sync strategy to use (mirrorLocal, mirrorRemote, preferLocal, preferRemote, preferNewest, keepAll)
  -l, --label-selector string      Comma separated key=value selector list (e.g. release=test)
      --local-path string          Local path to use (Default is current directory
      --no-watch                   Synchronizes local and remote and then stops
      --pick                       Select a pod (default true)
      --pod string                 Pod to sync to
      --polling bool               If polling should be used to detect file changes in the container
      --upload-only                If set DevSpace will only upload files
      --verbose                    Shows every file that is synced
      --wait                       Wait for the pod(s) to start if they are not running

```


## Global & Inherited Flags

```
      --config string                The devspace config file to use
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true will ignore all profile activations
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems (default 180)
      --kube-context string          The kubernetes context to use
  -n, --namespace string             The kubernetes namespace to use
      --no-warn                      If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --profile-parent strings       One or more profiles that should be applied before the specified profile (e.g. devspace dev --profile-parent=base1 --profile-parent=base2 --profile=my-profile)
      --profile-refresh              If true will pull and re-download profile parent sources
      --restore-vars                 If true will restore the variables from kubernetes before loading the config
      --save-vars                    If true will save the variables to kubernetes after loading the config
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               DEPRECATED: Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
      --vars-secret string           The secret to restore/save the variables from/to, if --restore-vars or --save-vars is enabled (default "devspace-vars")
```
