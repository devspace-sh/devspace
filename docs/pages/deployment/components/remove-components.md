---
title: Remove components
---

Run the following command to remove a component:
```bash
devspace remove deployment [deployment-name]
```

Before actually removing the component, DevSpace CLI will ask you the following question:
```bash
? Do you want to delete all deployment resources deployed?  [Use arrows to move, type to filter]
> yes
  no
```

> Deleting all resources deployed to Kubernetes before removing a component is very useful, so you do not end up with untracked resources which waste computing resources although they are not needed anymore.
