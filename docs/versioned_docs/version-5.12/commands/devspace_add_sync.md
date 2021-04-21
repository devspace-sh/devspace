---
title: "Command - devspace add sync"
sidebar_label: devspace add sync
---


Add a sync path

## Synopsis


```
devspace add sync [flags]
```

```
#######################################################
################# devspace add sync ###################
#######################################################
Add a sync path to this project's devspace.yaml

Example:
devspace add sync --local=app --container=/app
#######################################################
```


## Flags

```
      --container string        Absolute container path
      --exclude string          Comma separated list of paths to exclude (e.g. node_modules/,bin,*.exe)
  -h, --help                    help for sync
      --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --local string            Relative local path
```


## Global & Inherited Flags

```
      --config string            The devspace config file to use
      --debug                    Prints the stack trace if an error occurs
      --inactivity-timeout int   Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems (default 180)
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

