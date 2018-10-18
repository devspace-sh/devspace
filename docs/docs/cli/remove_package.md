---
title: devspace remove package
---

With `devspace remove package`, you can remove a package from a devspace deployment.  

`devspace remove package` deletes the specified packagename from the chart/requirements.yaml and executes the internal `helm dependency update` function.  

```
Usage:
  devspace remove package [flags]

Flags:
      --all                 Remove all packages
  -d, --deployment string   The deployment name to use
  -h, --help                help for package
```
