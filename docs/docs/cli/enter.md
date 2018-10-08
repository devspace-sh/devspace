---
title: devspace enter
---

Execute a command or start a new terminal in your devspace.  

```bash
Usage:
  devspace enter [flags]

Flags:
  -c, --container string   Container name within pod where to execute command
  -h, --help               help for enter

Examples: 
devspace enter
devspace enter bash
devspace enter echo 123
devspace enter -c myContainer
```
