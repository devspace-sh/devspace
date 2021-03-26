---
title: "Command - devspace deploy"
sidebar_label: devspace deploy
---


Deploy the project

## Synopsis


```
devspace deploy [flags]
```

```
#######################################################
################## devspace deploy ####################
#######################################################
Deploys the current project to a Space or namespace:

devspace deploy
devspace deploy -n some-namespace
devspace deploy --kube-context=deploy-context
#######################################################
```


## Flags

```
      --allow-cyclic           When enabled allows cyclic dependencies
      --build-sequential       Builds the images one after another instead of in parallel
      --dependency strings     Deploys only the specific named dependencies
      --deployments string     Only deploy a specifc deployment (You can specify multiple deployments comma-separated
  -b, --force-build            Forces to (re-)build every image
      --force-dependencies     Forces to re-evaluate dependencies (use with --force-build --force-deploy to actually force building & deployment of dependencies) (default true)
  -d, --force-deploy           Forces to (re-)deploy every deployment
  -h, --help                   help for deploy
      --skip-build             Skips building of images
      --skip-deploy            Skips deploying and only builds images
      --skip-push              Skips image pushing, useful for minikube deployment
      --skip-push-local-kube   Skips image pushing, if a local kubernetes environment is detected (default true)
      --timeout int            Timeout until deploy should stop waiting (default 120)
      --verbose-dependencies   Deploys the dependencies verbosely
      --wait                   If true will wait for pods to be running or fails after given timeout
```


## Global & Inherited Flags

```
      --config string            The devspace config file to use
      --debug                    Prints the stack trace if an error occurs
      --inactivity-timeout int   Minutes the current user is inactive (no mouse or keyboard interaction) until DevSpace will exit automatically. 0 to disable (default 180)
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

