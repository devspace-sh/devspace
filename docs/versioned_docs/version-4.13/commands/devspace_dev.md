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
- Use "devspace dev -i" for opening a terminal and overriding container entrypoint with sleep command
#######################################################
```


## Flags

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --build-sequential       Builds the images one after another instead of in parallel
      --deployments string     Only deploy a specific deployment (You can specify multiple deployments comma-separated
      --exit-after-deploy      Exits the command after building the images and deploying the project
  -b, --force-build            Forces to build every image
      --force-dependencies     Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies) (default true)
  -d, --force-deploy           Forces to deploy every deployment
  -h, --help                   help for dev
  -i, --interactive            Enable interactive mode for images (overrides entrypoint with sleep command) and start terminal proxy
      --open                   Open defined URLs in the browser, if defined (default true)
      --portforwarding         Enable port forwarding (default true)
      --skip-build             Skips building of images
  -x, --skip-pipeline          Skips build & deployment and only starts sync, portforwarding & terminal
      --skip-push              Skips image pushing, useful for minikube deployment
      --sync                   Enable code synchronization (default true)
  -t, --terminal               Open a terminal instead of showing logs
      --timeout int            Timeout until dev should stop waiting and fail (default 120)
      --ui                     Start the ui server (default true)
      --verbose-dependencies   Deploys the dependencies verbosely
      --verbose-sync           When enabled the sync will log every file change
      --wait                   If true will wait first for pods to be running or fails after given timeout
```


## Global & Inherited Flags

```
      --config string         The devspace config file to use
      --debug                 Prints the stack trace if an error occurs
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

