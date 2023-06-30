---
title: "devspace init --help"
sidebar_label: devspace init
---


Initializes DevSpace in the current folder

## Synopsis


```
devspace init [flags]
```

```
#######################################################
#################### devspace init ####################
#######################################################
Initializes a new devspace project within the current
folder. Creates a devspace.yaml as a starting point.
#######################################################
```


## Flags

```
      --context string      Context path to use for intialization
      --dockerfile string   Dockerfile to use for initialization (default "./Dockerfile")
  -h, --help                help for init
      --provider string     The cloud provider to use
  -r, --reconfigure         Change existing configuration
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

