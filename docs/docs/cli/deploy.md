---
title: devspace deploy
---

With `devspace deploy` the `devspace up` pipeline is only run once and sync, portforwarding and terminal are not started.

```
Usage:
  devspace deploy [flags]

Flags:
      --cloud-target string    When using a cloud provider, the target to use
      --config string          The devspace config file to load (default: '.devspace/config.yaml' (default "/.devspace/config.yaml")
      --docker-target string   The docker target to use for building
  -h, --help                   help for deploy
      --kube-context string    The kubernetes context to use for deployment
      --namespace string       The namespace to deploy to
      --switch-context         Switches the kube context to the deploy context (default true)

Examples:

devspace deploy --namespace=deploy
devspace deploy --namespace=deploy --docker-target=production
devspace deploy --kube-context=minikube --namespace=deploy
devspace deploy --config=.devspace/deploy.yaml
devspace deploy --cloud-target=production
```
