---
title: devspace add port
id: version-v3.5.18-port
original_id: port
---

```bash
#######################################################
################ devspace add port ####################
#######################################################
Add a new port mapping to your DevSpace configuration
(format is local:remote comma separated):
devspace add port 8080:80,3000
#######################################################

Usage:
  devspace add port [flags]

Flags:
  -h, --help                    help for port
      --label-selector string   Comma separated key=value label-selector list (e.g. release=test)
      --namespace string        Namespace to use
      --selector string         Name of a selector defined in your DevSpace config
```
