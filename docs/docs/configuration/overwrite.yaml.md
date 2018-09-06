---
title: /.devspace/overwrite.yaml
---

This is an example of the [.devspace/overwrite.yaml](#)

```yaml
version: v1
release:
  name: devspace-docs
  namespace: devspace-docs
  latestBuild: "2018-08-27T17:32:42.5568625+02:00"
  latestImage: 10.99.117.105:5000/devspace-docs:vkY31ug8IQ
registry:
  release:
    name: devspace-registry
    namespace: devspace-docs
  user:
    username: user-xNAsk
    password: tY2XpapTMUjt
cluster:
  tillerNamespace: devspace-docs
  useKubeConfig: true
```

The [.devspace/overwrite.yaml](#) is defined for every developer who wants to use a DevSpace for working on the respective project. This file should **never** be checked into a version control system. Therefore, the DevSpace CLI will automatically create a [.gitignore](#) file within [.devspace/](#) that tells git not to version this file.

**Note: You can easily re-configure your DevSpace by running `devspace init -r`. Therefore, changing this file manually is highly discouraged.**
