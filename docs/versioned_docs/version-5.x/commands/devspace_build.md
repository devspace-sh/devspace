---
title: "Command - devspace build"
sidebar_label: devspace build
---


Builds all defined images and pushes them

## Synopsis


```
devspace build [flags]
```

```
#######################################################
################## devspace build #####################
#######################################################
Builds all defined images and pushes them
#######################################################
```


## Flags

```
      --build-sequential            Builds the images one after another instead of in parallel
      --dependency strings          Builds only the specific named dependencies
  -b, --force-build                 Forces to build every image
      --force-dependencies          Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies) (default true)
  -h, --help                        help for build
      --max-concurrent-builds int   The maximum number of image builds built in parallel (0 for infinite)
      --skip-dependency strings     Skips building the following dependencies
      --skip-push                   Skips image pushing, useful for minikube deployment
      --skip-push-local-kube        Skips image pushing, if a local kubernetes environment is detected
  -t, --tag strings                 Use the given tag for all built images
      --verbose-dependencies        Builds the dependencies verbosely (default true)
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

