---
title: devspace remove package
---

With `devspace remove package`, you can remove a package from the devspace.  

`devspace remove package` deletes the specified packagename from the chart/requirements.yaml and executes the internal `helm dependency update` function.  

```
Usage:
  devspace remove package [flags]

Flags:
      --all    Remove all packages
  -h, --help   help for package
```
