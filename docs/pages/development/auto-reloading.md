---
title: Configure auto-reloading
---

There are certain use cases where you want to redeploy the whole application instead of only syncing certain files into a container. DevSpace provides you the options to specify special paths that are watched during `devspace dev` and any change to such a path will trigger a redeploy.  

A minimal configuration for such behavior can look like this:
```yaml
dev:
  autoReload:
    paths:
    # Any glob path can be here, in this case this means all files and folders
    # in the current project
    - ./**
```

With this configuration, DevSpace will rebuild images (if necessary) and redeploy deployments if a certain path has changed. You can also take a look at the [redeploy-instead-of-hot-reload](https://github.com/devspace-cloud/devspace/tree/master/examples/redeploy-instead-of-hot-reload) to see a working example.  
