---
title: "Command - devspace run"
sidebar_label: devspace run
---


Run executes a predefined command

## Synopsis


```
devspace run [flags]
```

```
#######################################################
##################### devspace run ####################
#######################################################
Run executes a predefined command from the devspace.yaml

Examples:
devspace run mycommand --myarg 123
devspace run mycommand2 1 2 3
devspace --dependency my-dependency run any-command --any-command-flag
#######################################################
```


## Flags

```
      --dependency string   Run a command from a specific dependency
  -h, --help                help for run
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

