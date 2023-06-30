---
title: "devspace logs --help"
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
Prints the last log of a pod container and attachs 
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
      --image-selector string   The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --lines int               Max amount of lines to print from the last log (default 200)
      --pick                    Select a pod (default true)
      --pod string              Pod to print the logs of
      --wait                    Wait for the pod(s) to start if they are not running
```


## Global & Inherited Flags

```
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true will ignore all profile activations
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems
      --kube-context string          The kubernetes context to use
      --kubeconfig string            The kubeconfig path to use
  -n, --namespace string             The kubernetes namespace to use
      --no-colors                    Do not show color highlighting in log output. This avoids invisible output with different terminal background colors
      --no-warn                      If true does not show any warning when deploying into a different namespace or kube-context than before
      --override-name string         If specified will override the DevSpace project name provided in the devspace.yaml
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

