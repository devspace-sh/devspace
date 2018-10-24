---
title: devspace down
---

Run `devspace down` to shutdown your DevSpace. Stops your DevSpace by removing the release via helm (if deployment method is helm) or by running kubectl delete over the manifests. If you want to remove all DevSpace related data from your project, use: devspace reset.

```bash
Usage:
  devspace down [flags]

Flags:
  -h, --help   help for down
```
