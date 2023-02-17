---
title: "devspace dev --help"
sidebar_label: devspace dev
---


Starts the development mode

## Synopsis


```
devspace dev [flags]
```

```
#######################################################
################### devspace dev ######################
#######################################################
Starts your project in development mode
#######################################################
```


## Flags

```
      --build-sequential            Builds the images one after another instead of in parallel
      --dependency strings          Deploys only the specified named dependencies
  -b, --force-build                 Forces to build every image
  -d, --force-deploy                Forces to deploy every deployment
      --force-purge                 Forces to purge every deployment even though it might be in use by another DevSpace project
  -h, --help                        help for dev
      --max-concurrent-builds int   The maximum number of image builds built in parallel (0 for infinite)
      --pipeline string             The pipeline to execute (default "dev")
      --render                      If true will render manifests and print them instead of actually deploying them
      --sequential-dependencies     If set set true dependencies will run sequentially
      --show-ui                     Shows the ui server
      --skip-build                  Skips building of images
      --skip-dependency strings     Skips the following dependencies for deployment
      --skip-deploy                 If enabled will skip deploying
      --skip-push                   Skips image pushing, useful for minikube deployment
      --skip-push-local-kube        Skips image pushing, if a local kubernetes environment is detected (default true)
  -t, --tag strings                 Use the given tag for all built images
```


## Global & Inherited Flags

```
      --debug                        Prints the stack trace if an error occurs
      --disable-profile-activation   If true will ignore all profile activations
      --inactivity-timeout int       Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable. Only supported on windows and mac operating systems
      --kube-context string          The kubernetes context to use
      --kubeconfig string            The kubeconfig path to use
  -n, --namespace string             The kubernetes namespace to use
      --no-warn                      If true does not show any warning when deploying into a different namespace or kube-context than before
      --override-name string         If specified will override the devspace.yaml name
  -p, --profile strings              The DevSpace profiles to apply. Multiple profiles are applied in the order they are specified
      --silent                       Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context               Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings                  Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

