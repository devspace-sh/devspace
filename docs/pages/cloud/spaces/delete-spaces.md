---
title: Delete Spaces
---

If you just want to delete the `kubectl` context of a space you can run:
```bash
devspace remove context [SPACE_NAME]
```

This command will remove the kube context for the given space from your `kubectl` config. If you want to delete the complete namespace and all resources in it you can run:
```bash
devspace remove space [SPACE_NAME]
```

This will remove all deployed resources from the space and delete the isolated namespace. In addition this does also delete the kube context for the space used locally.

> **Warning: Deleting a Space will also delete all the persistent volumes defined in this Space. Your data will be lost forever and cannot be recovered again.**
