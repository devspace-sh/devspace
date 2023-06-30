---
title: "devspace enter --help"
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
      --image-selector string   The image to search a pod for (e.g. nginx, nginx:latest, ${runtime.images.app}, nginx:${runtime.images.app.tag})
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --pick                    Select a pod / container if multiple are found (default true)
      --pod string              Pod to open a shell to
      --reconnect               Will reconnect the terminal if an unexpected return code is encountered
      --screen                  Use a screen session to connect
      --screen-session string   The screen session to create or connect to (default "enter")
      --tty                     If to use a tty to start the command (default true)
      --wait                    Wait for the pod(s) to start if they are not running
      --workdir string          The working directory where to open the terminal or execute the command
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

