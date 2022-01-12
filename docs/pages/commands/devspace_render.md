---
title: "Command - devspace render"
sidebar_label: devspace render
---


Render builds all defined images and shows the yamls that would be deployed

## Synopsis


```
devspace render [flags]
```

```
#######################################################
################## devspace render #####################
#######################################################
Builds all defined images and shows the yamls that would
be deployed via helm and kubectl, but skips actual 
deployment.
#######################################################
```


## Flags

```
      --build-sequential            Builds the images one after another instead of in parallel
      --dependency strings          Renders only the specific named dependencies
      --deployments string          Only deploy a specific deployment (You can specify multiple deployments comma-separated
  -b, --force-build                 Forces to build every image
  -h, --help                        help for render
      --max-concurrent-builds int   The maximum number of image builds built in parallel (0 for infinite)
      --skip-build                  Skips image building
      --skip-dependencies           Skips rendering the dependencies
      --skip-dependency strings     Skips rendering the following dependencies
      --skip-push                   Skips image pushing, useful for minikube deployment
      --skip-push-local-kube        Skips image pushing, if a local kubernetes environment is detected (default true)
  -t, --tag strings                 Use the given tag for all built images
      --verbose-dependencies        Builds the dependencies verbosely
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

