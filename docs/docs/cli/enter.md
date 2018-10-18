---
title: devspace enter
---

Execute a command or start a new terminal in your devspace.  

```bash
Usage:
  devspace enter [flags]

Flags:
  -c, --container string        Container name within pod where to execute command
  -h, --help                    help for enter
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
  -n, --namespace string        Namespace where to select pods

Examples: 
devspace enter
devspace enter bash
devspace enter -c myContainer
devspace enter echo 123 -n my-namespace
devspace enter bash -l release=test
```
