---
title: devspace remove sync
sidebar_label: sync
---

```bash
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

Usage:
  devspace remove sync [flags]

Flags:
      --all                     Remove all configured sync paths
      --container string        Absolute container path to remove
  -h, --help                    help for sync
      --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --local string            Relative local path to remove
```
