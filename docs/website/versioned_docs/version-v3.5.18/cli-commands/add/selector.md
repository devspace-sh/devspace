---
title: devspace add selector
id: version-v3.5.18-selector
original_id: selector
---

```bash
#######################################################
############# devspace add selector ###################
#######################################################
Add a new selector to your DevSpace configuration

Examples:
devspace add selector my-selector --namespace=my-namespace
devspace add selector my-selector --label-selector=environment=production,tier=frontend
#######################################################

Usage:
  devspace add selector [flags]

Flags:
  -h, --help                    help for selector
      --label-selector string   The label-selector of the selector
      --namespace string        The namespace of the selector
```
