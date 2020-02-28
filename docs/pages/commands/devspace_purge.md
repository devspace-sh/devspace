---
title: "Command - devspace purge"
sidebar_label: devspace purge
---


Delete deployed resources

## Synopsis


```
devspace purge [flags]
```

```
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################
```


## Flags

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --dependencies           When enabled purges the dependencies as well
      --dependency strings     Purges only the specific named dependencies
  -d, --deployments string     The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)
  -h, --help                   help for purge
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

