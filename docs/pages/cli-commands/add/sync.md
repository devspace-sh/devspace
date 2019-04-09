---
title: devspace add sync
---

```bash
#######################################################
################# devspace add sync ###################
#######################################################
Add a sync path to the DevSpace configuration

How to use:
devspace add sync --local=app --container=/app
#######################################################

Usage:
  devspace add sync [flags]

Flags:
      --container string        Absolute container path
      --exclude string          Comma separated list of paths to exclude (e.g. node_modules/,bin,*.exe)
  -h, --help                    help for sync
      --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --local string            Relative local path
      --namespace string        Namespace to use
      --selector string         Name of a selector defined in your DevSpace config
```
