---
title: devspace list spaces
---

```bash
#######################################################
############### devspace list spaces ##################
#######################################################
List all user cloud spaces

Example:
devspace list spaces
devspace list spaces --cluster my-cluster
devspace list spaces --all
#######################################################

Usage:
  devspace list spaces [flags]

Flags:
      --all               List all spaces the user has access to in all clusters (not only created by the user)
      --cluster string    List all spaces in a certain cluster
  -h, --help              help for spaces
      --name string       Space name to show (default: all)
      --provider string   Cloud Provider to use
```
