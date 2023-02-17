---
title: "devspace sync --help"
sidebar_label: devspace sync
---


Starts a bi-directional sync between the target container and the local path

## Synopsis


```
devspace sync [flags]
```

```
#############################################################################
################### devspace sync ###########################################
#############################################################################
Starts a bi-directional(default) sync between the target container path
and local path:

devspace sync --path=.:/app # localPath is current dir and remotePath is /app
devspace sync --path=.:/app --image-selector nginx:latest
devspace sync --path=.:/app --exclude=node_modules,test
devspace sync --path=.:/app --pod=my-pod --container=my-container
#############################################################################
```


## Flags

```
  -c, --container string           Container name within pod where to sync to
      --download-on-initial-sync   DEPRECATED: Downloads all locally non existing remote files in the beginning (default true)
      --download-only              If set DevSpace will only download files
  -e, --exclude strings            Exclude directory from sync
  -h, --help                       help for sync
      --image-selector string      The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})
      --initial-sync string        The initial sync strategy to use (mirrorLocal, mirrorRemote, preferLocal, preferRemote, preferNewest, keepAll)
  -l, --label-selector string      Comma separated key=value selector list (e.g. release=test)
      --no-watch                   Synchronizes local and remote and then stops
      --path string                Path to use (Default is current directory). Example: ./local-path:/remote-path or local-path:.
      --pick                       Select a pod (default true)
      --pod string                 Pod to sync to
      --polling                    If polling should be used to detect file changes in the container
      --upload-only                If set DevSpace will only upload files
      --wait                       Wait for the pod(s) to start if they are not running (default true)
```


## Global & Inherited Flags

```
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true will ignore all profile activations
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems
      --kube-context string          The kubernetes context to use
      --kubeconfig string            The kubeconfig path to use
  -n, --namespace string             The kubernetes namespace to use
      --no-warn                      If true does not show any warning when deploying into a different namespace or kube-context than before
      --override-name string         If specified will override the devspace.yaml name
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

