---
title: "devspace set var --help"
sidebar_label: devspace set var
---


Sets a variable

## Synopsis


```
devspace set var [flags]
```

```
#######################################################
################# devspace set var ####################
#######################################################
Sets a specific variable 

Examples:
devspace set var key=value
devspace set var key=value key2=value2
#######################################################
```


## Flags

```
  -h, --help        help for var
      --overwrite   If true will overwrite the variables value even if its set already (default true)
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

