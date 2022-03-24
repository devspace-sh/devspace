---
title: "Command - devspace"
sidebar_label: devspace
---

## devspace

Welcome to the DevSpace!

### Synopsis

DevSpace accelerates developing, deploying and debugging applications with Docker and Kubernetes. To get started using DevSpace, run the following command in one of your projects:
	
		devspace init

### Options

When you run the `init` command, you will see a number of additional commands and options you can use with DevSpace.

```
      --config string                The devspace config file to use
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true, will ignore all profile activations
  -h, --help                         help page for devspace
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems (default 180)
      --kube-context string          The kubernetes context to use
  -n, --namespace string             The kubernetes namespace to use
      --no-warn                      If true, does not show any warning when deploying into a different namespace or kube-context
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --profile-parent strings       One or more profiles that should be applied before the specified profile (e.g. devspace dev --profile-parent=base1 --profile-parent=base2 --profile=my-profile)
      --profile-refresh              If true, will pull and re-download profile parent sources
      --restore-vars                 If true, will restore the variables from kubernetes before loading the config
      --save-vars                    If true, will save the variables to kubernetes after loading the config
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               DEPRECATED: Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
      --vars-secret string           The secret used to restore/save the variables from/to, if --restore-vars or --save-vars is enabled (default "devspace-vars")
```

```

```


## Flags
## Global & Inherited Flags
