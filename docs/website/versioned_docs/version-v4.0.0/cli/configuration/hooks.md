---
title: Defining Hooks for the Build & Deployment Process
sidebar_label: Hooks
id: version-v4.0.0-hooks
original_id: hooks
---

DevSpace allows you to define execution of custom commands during certain lifecycle events. This makes it possible to customize the deployment and development process with DevSpace.

> For a complete example take a look at [hooks](https://github.com/devspace-cloud/devspace/tree/master/examples/hooks)

## Define a hook

To define a hook modify your `devspace.yaml` like this:

```yaml
hooks:
  - command: echo
    args:
      - before image building
    when:
      before:
        images: all
```

This tells DevSpace to execute the command `echo before image building` before any image will be built. You are able to define hooks for the following life cycle events:
- **before image building**: Will be executed before building any images. Value: `when.before.images: all`
- **after image building**: Will be executed after images have been successfully built. Value: `when.after.images: all`
- **before deploying**: Will be executed before any deployment is deployed. Value: `when.before.deployments: all`
- **after deploying**: Will be executed after all deployments are deployed. Value: `when.after.deployments: all`
- **before certain deployment**: Will be executed before a certain deployment is deployed.  Value: `when.before.deployments: my-deployment`
- **after certain deployment**: Will be executed after a certain deployment is deployed.  Value: `when.after.deployments: my-deployment`

> If any hook returns a non zero exit code, DevSpace will abort and print an error message!
