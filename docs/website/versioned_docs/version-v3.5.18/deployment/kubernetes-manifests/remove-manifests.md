---
title: Remove Kubernetes manifests
id: version-v3.5.18-remove-manifests
original_id: remove-manifests
---

Run the following command to remove a manifest deployment:
```bash
devspace remove deployment [deployment-name]
```

Before actually removing the manifest deployment, DevSpace CLI will ask you the following question:
```bash
? Do you want to delete all deployment resources deployed?  [Use arrows to move, type to filter]
> yes
  no
```

> Deleting all resources deployed to Kubernetes before removing a manifest deployment is very useful, so you do not end up with untracked resources which waste computing resources although they are not needed anymore.
