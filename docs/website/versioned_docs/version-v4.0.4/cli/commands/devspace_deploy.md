---
title: Command - devspace deploy
sidebar_label: devspace deploy
id: version-v4.0.4-devspace_deploy
original_id: devspace_deploy
---


Deploy the project

## Synopsis


```
devspace deploy [flags]
```

```
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy --namespace=deploy
devspace deploy --namespace=deploy
devspace deploy --kube-context=deploy-context
#######################################################
```
## Options

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --build-sequential       Builds the images one after another instead of in parallel
      --deployments string     Only deploy a specifc deployment (You can specify multiple deployments comma-separated
  -b, --force-build            Forces to (re-)build every image
      --force-dependencies     Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)
  -d, --force-deploy           Forces to (re-)deploy every deployment
  -h, --help                   help for deploy
      --skip-build             Skips building of images
      --skip-push              Skips image pushing, useful for minikube deployment
      --verbose-dependencies   Deploys the dependencies verbosely
```

### Options inherited from parent commands

```
      --debug                 Prints the stack trace if an error occurs
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```
