---
title: "Command - devspace build"
sidebar_label: devspace build
---


Builds all defined images and pushes them

## Synopsis


```
devspace build [flags]
```

```
#######################################################
################## devspace build #####################
#######################################################
Builds all defined images and pushes them
#######################################################
```
## Options

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --build-sequential       Builds the images one after another instead of in parallel
  -b, --force-build            Forces to build every image
      --force-dependencies     Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies)
  -h, --help                   help for build
      --skip-push              Skips image pushing, useful for minikube deployment
      --verbose-dependencies   Builds the dependencies verbosely
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

## See Also
