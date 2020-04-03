---
title: "Command - devspace logs"
sidebar_label: devspace logs
---


Prints the logs of a pod and attaches to it

## Synopsis


```
devspace logs [flags]
```

```
#######################################################
#################### devspace logs ####################
#######################################################
Logs prints the last log of a pod container and attachs 
to it

Example:
devspace logs
devspace logs --namespace=mynamespace
#######################################################
```


## Flags

```
  -c, --container string        Container name within pod where to execute command
  -f, --follow                  Attach to logs afterwards
  -h, --help                    help for logs
      --image string            Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --lines int               Max amount of lines to print from the last log (default 200)
      --pick                    Select a pod
      --pod string              Pod to print the logs of
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

