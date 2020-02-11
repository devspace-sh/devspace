---
title: devspace enter
id: version-v3.5.18-enter
original_id: enter
---

```bash
#######################################################
################## devspace enter #####################
#######################################################
Execute a command or start a new terminal in your
devspace:

devspace enter
devspace enter -p # Select pod to enter
devspace enter bash
devspace enter -s my-selector
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
#######################################################

Usage:
  devspace enter [flags]

Flags:
  -c, --container string        Container name within pod where to execute command
  -h, --help                    help for enter
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
  -n, --namespace string        Namespace where to select pods
  -p, --pick                    Select a pod
      --pod string              Pod to open a shell to
  -s, --selector string         Selector name (in config) to select pod/container for terminal
      --switch-context          Switch kubectl context to the DevSpace context
```
