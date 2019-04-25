---
title: devspace reset key
---

```bash
#######################################################
################### devspace sync #####################
#######################################################
Starts a bi-directionaly sync between the target container
and the current path:

devspace sync
devspace sync --exclude=node_modules --exclude=test
devspace sync --pod=my-pod --container=my-container
devspace sync --container-path=/my-path
#######################################################

Usage:
  devspace sync [flags]

Flags:
  -c, --container string        Container name within pod where to execute command
      --container-path string   Container path to use (Default is working directory)
  -e, --exclude strings         Exclude directory from sync
  -h, --help                    help for sync
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
  -n, --namespace string        Namespace where to select pods
  -p, --pick                    Select a pod to stream logs from
      --pod string              Pod to open a shell to
  -s, --selector string         Selector name (in config) to select pod/container for terminal
```
