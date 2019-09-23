---
title: "Command - devspace attach"
sidebar_label: devspace attach
---


Attaches to a container

## Synopsis


```
devspace attach [flags]
```

```
#######################################################
################# devspace attach #####################
#######################################################
Attaches to a running container

devspace attach
devspace attach --pick # Select pod to enter
devspace attach -c my-container
devspace attach -n my-namespace
#######################################################
```
## Options

```
  -c, --container string        Container name within pod where to execute command
  -h, --help                    help for attach
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --pick                    Select a pod
      --pod string              Pod to open a shell to
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

