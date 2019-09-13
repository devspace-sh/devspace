---
title: devspace remove selector
id: version-v3.5.18-selector
original_id: selector
---

```bash
#######################################################
############ devspace remove selector #################
#######################################################
Removes one, multiple or all selectors from a devspace.
If the argument is specified, the selector with that name will be deleted.
If more than one condition for deletion is specified, all selectors that match at least one of the conditions will be deleted.

Examples:
devspace remove selector my-selector
devspace remove selector --namespace=my-namespace --label-selector=environment=production,tier=frontend
devspace remove selector --all
#######################################################

Usage:
  devspace remove selector [flags]

Flags:
      --all                     Remove all selectors
  -h, --help                    help for selector
      --label-selector string   Label-selector of the selector
      --namespace string        Namespace of the selector
```
