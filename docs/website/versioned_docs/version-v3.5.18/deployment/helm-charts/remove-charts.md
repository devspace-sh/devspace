---
title: Remove Helm charts
id: version-v3.5.18-remove-charts
original_id: remove-charts
---

Run the following command to remove a Helm chart from your deployments:
```bash
devspace remove deployment [deployment-name]
```

Before actually removing the deployment, DevSpace CLI will ask you the following question:
```bash
? Do you want to delete all deployment resources deployed?  [Use arrows to move, type to filter]
> yes
  no
```

> Deleting all resources deployed to Kubernetes before removing a Helm chart deployment is very useful, so you do not end up with untracked resources which waste computing resources although they are not needed anymore.
