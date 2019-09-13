---
title: devspace purge
id: version-v3.5.18-purge
original_id: purge
---

```bash
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge -d my-deployment
#######################################################

Usage:
  devspace purge [flags]

Flags:
  -d, --deployments string   The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)
  -h, --help                 help for purge
```
