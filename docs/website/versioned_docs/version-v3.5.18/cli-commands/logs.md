---
title: devspace logs
id: version-v3.5.18-logs
original_id: logs
---

```bash
#######################################################
#################### devspace logs ####################
#######################################################
Logs prints the last log of a pod container and attachs
to it

Example:
devspace logs
devspace logs --namespace=mynamespace
#######################################################

Usage:
  devspace logs [flags]

Flags:
  -c, --container string        Container name within pod where to execute command
  -f, --follow                  Attach to logs afterwards
  -h, --help                    help for logs
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --lines int               Max amount of lines to print from the last log (default 200)
  -n, --namespace string        Namespace where to select pods
  -p, --pick                    Select a pod
      --pod string              Pod to print the logs of
  -s, --selector string         Selector name (in config) to select pod/container for terminal
```
