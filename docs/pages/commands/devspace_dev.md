---
title: "Command - devspace dev"
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
Starts your project in development mode:
1. Builds your Docker images and override entrypoints if specified
2. Deploys the deployments via helm or kubectl
3. Forwards container ports to the local computer
4. Starts the sync client
5. Streams the logs of deployed containers

Open terminal instead of logs:
- Use "devspace dev -t" for opening a terminal
#######################################################
```


## Flags

```
      --build-sequential            Builds the images one after another instead of in parallel
      --dependency strings          Deploys only the specified named dependencies
      --deployments string          Only deploy a specific deployment (You can specify multiple deployments comma-separated
      --exit-after-deploy           Exits the command after building the images and deploying the project
  -b, --force-build                 Forces to build every image
      --force-dependencies          Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies) (default true)
  -d, --force-deploy                Forces to deploy every deployment
  -h, --help                        help for dev
  -i, --interactive                 DEPRECATED: DO NOT USE ANYMORE
      --max-concurrent-builds int   The maximum number of image builds built in parallel (0 for infinite)
      --open                        Open defined URLs in the browser, if defined (default true)
      --portforwarding              Enable port forwarding (default true)
      --print-sync                  If enabled will print the sync log to the terminal
      --skip-build                  Skips building of images
      --skip-dependency strings     Skips the following dependencies for deployment
  -x, --skip-pipeline               Skips build & deployment and only starts sync, portforwarding & terminal
      --skip-push                   Skips image pushing, useful for minikube deployment
      --skip-push-local-kube        Skips image pushing, if a local kubernetes environment is detected (default true)
      --sync                        Enable code synchronization (default true)
  -t, --terminal                    Open a terminal instead of showing logs
      --terminal-reconnect          Will try to reconnect the terminal if an unexpected exit code was encountered (default true)
      --timeout int                 Timeout until dev should stop waiting and fail (default 120)
      --ui                          Start the ui server (default true)
      --ui-port int                 The port to use when opening the ui server
      --verbose-dependencies        Deploys the dependencies verbosely (default true)
      --verbose-sync                When enabled the sync will log every file change
      --wait                        If true will wait first for pods to be running or fails after given timeout
      --workdir string              The working directory where to open the terminal or execute the command
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

