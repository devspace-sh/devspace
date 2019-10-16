---
title: Command - devspace analyze
sidebar_label: devspace analyze
id: version-v4.0.4-devspace_analyze
original_id: devspace_analyze
---


Analyzes a kubernetes namespace and checks for potential problems

## Synopsis


```
devspace analyze [flags]
```

```
#######################################################
################## devspace analyze ###################
#######################################################
Analyze checks a namespaces events, replicasets, services
and pods for potential problems

Example:
devspace analyze
devspace analyze --namespace=mynamespace
#######################################################
```
## Options

```
  -h, --help   help for analyze
      --wait   Wait for pods to get ready if they are just starting (default true)
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
