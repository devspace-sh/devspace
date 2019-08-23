---
title: devspace deploy
---

```bash
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy --namespace=deploy
devspace deploy --namespace=deploy
devspace deploy --kube-context=deploy-context
#######################################################

Usage:
  devspace deploy [flags]

Flags:
      --docker-target string   The docker target to use for building
  -b, --force-build            Forces to (re-)build every image
  -d, --force-deploy           Forces to (re-)deploy every deployment
  -h, --help                   help for deploy
      --kube-context string    The kubernetes context to use for deployment
      --namespace string       The namespace to deploy to
      --switch-context         Switches the kube context to the deploy context
```
