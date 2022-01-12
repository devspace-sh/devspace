---
title: "Command - devspace deploy"
sidebar_label: devspace deploy
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
devspace deploy -n some-namespace
devspace deploy --kube-context=deploy-context
#######################################################
```


## Flags

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --build-sequential       Builds the images one after another instead of in parallel
      --dependency strings     Deploys only the specific named dependencies
      --deployments string     Only deploy a specific deployment (You can specify multiple deployments comma-separated
  -b, --force-build            Forces to (re-)build every image
      --force-dependencies     Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies) (default true)
  -d, --force-deploy           Forces to (re-)deploy every deployment
  -h, --help                   help for deploy
      --skip-build             Skips building of images
      --skip-push              Skips image pushing, useful for minikube deployment
      --timeout int            Timeout until deploy should stop waiting (default 120)
      --verbose-dependencies   Deploys the dependencies verbosely
      --wait                   If true will wait for pods to be running or fails after given timeout
```


## Global & Inherited Flags

```
      --config string         The devspace config file to use
      --debug                 Prints the stack trace if an error occurs
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

