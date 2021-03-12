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
      --pick                    Select a pod (default true)
      --pod string              Pod to print the logs of
      --wait                    Wait for the pod(s) to start if they are not running
```


## Global & Inherited Flags

```
      --config string            The devspace config file to use
      --debug                    Prints the stack trace if an error occurs
      --inactivity-timeout int   Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable (default 180)
      --kube-context string      The kubernetes context to use
  -n, --namespace string         The kubernetes namespace to use
      --no-warn                  If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string           The devspace profile to use (if there is any)
      --profile-parent strings   One or more profiles that should be applied before the specified profile (e.g. devspace dev --profile-parent=base1 --profile-parent=base2 --profile=my-profile)
      --profile-refresh          If true will pull and re-download profile parent sources
      --restore-vars             If true will restore the variables from kubernetes before loading the config
      --save-vars                If true will save the variables to kubernetes after loading the config
      --silent                   Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context           Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings              Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
      --vars-secret string       The secret to restore/save the variables from/to, if --restore-vars or --save-vars is enabled (default "devspace-vars")
```

