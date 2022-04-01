---
title: "Command - devspace render"
sidebar_label: devspace render
---


Render builds all defined images and shows the yamls that would be deployed

## Synopsis


```
devspace render [flags]
```

```
#######################################################
################## devspace render #####################
#######################################################
Builds all defined images and shows the yamls that would
be deployed via helm and kubectl, but skips actual 
deployment.
#######################################################
```


## Flags

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --build-sequential       Builds the images one after another instead of in parallel
      --dependency strings     Renders only the specific named dependencies
      --deployments string     Only deploy a specific deployment (You can specify multiple deployments comma-separated
  -b, --force-build            Forces to build every image
  -h, --help                   help for render
      --show-logs              Shows the build logs
      --skip-build             Skips image building
      --skip-dependencies      Skips rendering the dependencies
      --skip-push              Skips image pushing, useful for minikube deployment
  -t, --tag strings            Use the given tag for all built images
      --verbose-dependencies   Builds the dependencies verbosely
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

