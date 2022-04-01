---
title: "Command - devspace enter"
sidebar_label: devspace enter
---


Open a shell to a container

## Synopsis


```
devspace enter [flags]
```

```
#######################################################
################## devspace enter #####################
#######################################################
Execute a command or start a new terminal in your 
devspace:

devspace enter
devspace enter --pick # Select pod to enter
devspace enter bash
devspace enter -c my-container
devspace enter bash -n my-namespace
devspace enter bash -l release=test
devspace enter bash --image-selector nginx:latest
devspace enter bash --image-selector "${runtime.images.app.image}:${runtime.images.app.tag}"
#######################################################
```


## Flags

```
  -c, --container string        Container name within pod where to execute command
  -h, --help                    help for enter
      --image string            Image is the config name of an image to select in the devspace config (e.g. 'default'), it is NOT a docker image like myuser/myimage
      --image-selector string   The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --pick                    Select a pod / container if multiple are found (default true)
      --pod string              Pod to open a shell to
      --reconnect               Will reconnect the terminal if an unexpected return code is encountered
      --wait                    Wait for the pod(s) to start if they are not running
      --workdir string          The working directory where to open the terminal or execute the command
```


## Global & Inherited Flags

```
      --config string                The devspace config file to use
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true will ignore all profile activations
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems (default 180)
      --kube-context string          The kubernetes context to use
  -n, --namespace string             The kubernetes namespace to use
      --no-warn                      If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --profile-parent strings       One or more profiles that should be applied before the specified profile (e.g. devspace dev --profile-parent=base1 --profile-parent=base2 --profile=my-profile)
      --profile-refresh              If true will pull and re-download profile parent sources
      --restore-vars                 If true will restore the variables from kubernetes before loading the config
      --save-vars                    If true will save the variables to kubernetes after loading the config
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               DEPRECATED: Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
      --vars-secret string           The secret to restore/save the variables from/to, if --restore-vars or --save-vars is enabled (default "devspace-vars")
```

