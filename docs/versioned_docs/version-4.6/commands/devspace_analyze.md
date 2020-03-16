---
title: "Command - devspace analyze"
sidebar_label: devspace analyze
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


## Flags

```
  -h, --help          help for analyze
      --patient       If true, analyze will ignore failing pods and events until every deployment, statefulset, replicaset and pods are ready or the timeout is reached
      --timeout int   Timeout until analyze should stop waiting (default 120)
      --wait          Wait for pods to get ready if they are just starting (default true)
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

