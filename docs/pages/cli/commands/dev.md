---
title: devspace dev
---

```bash
#######################################################
################### devspace dev ######################
#######################################################
Starts your project in development mode:
1. Builds your Docker images and override entrypoints if specified
2. Deploys the deployments via helm or kubectl
3. Forwards container ports to the local computer
4. Starts the sync client
5. Enters the container shell
#######################################################

Usage:
  devspace dev [flags]

Flags:
  -c, --container string        Container name where to open the shell
      --exit-after-deploy       Exits the command after building the images and deploying the project
  -b, --force-build             Forces to build every image
  -d, --force-deploy            Forces to deploy every deployment
  -h, --help                    help for dev
      --init-registries         Initialize registries (and install internal one) (default true)
  -l, --label-selector string   Comma separated key=value selector list to use for terminal (e.g. release=test)
  -n, --namespace string        Namespace where to select pods for terminal
      --portforwarding          Enable port forwarding (default true)
  -s, --selector string         Selector name (in config) to select pods/container for terminal
  -x, --skip-pipeline           Skips build & deployment and only starts sync, portforwarding & terminal
      --switch-context          Switch kubectl context to the DevSpace context
      --sync                    Enable code synchronization (default true)
      --terminal                Enable terminal (true or false) (default true)
      --verbose-sync            When enabled the sync will log every file change
```
