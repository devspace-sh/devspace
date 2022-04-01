---
title: "Command - devspace purge"
sidebar_label: devspace purge
---


Delete deployed resources

## Synopsis


```
devspace purge [flags]
```

```
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################
```


## Flags

```
  -a, --all                       When enabled purges the dependencies as well (default true)
      --dependencies              DEPRECATED: Please use --all instead
      --dependency strings        Purges only the specific named dependencies
  -d, --deployments string        The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)
  -h, --help                      help for purge
      --skip-dependency strings   Skips the following dependencies from purging
      --verbose-dependencies      Builds the dependencies verbosely (default true)
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

