---
title: devspace analyze
---

```js
#######################################################
################## devspace analyze ###################
#######################################################
Analyze checks a namespaces events, replicasets, services
and pods for potential problems

Example:
devspace analyze
devspace analyze --namespace=mynamespace
#######################################################

Usage:
  devspace analyze [flags]

Flags:
  -h, --help               help for analyze
  -n, --namespace string   The kubernetes namespace to analyze
      --wait               Wait for pods to get ready if they are just starting (default true)
```
