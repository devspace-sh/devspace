---
title: devspace up
---

With `devspace up`, the defined images are build, deployments are deployed and services started.  

The command will do the following:  

1. Build the specified images using docker or kaniko
2. Push the built images to the corresponding registries (either to a local or remote registry)
3. Deploy the configured deployments via helm or kubectl
4. Establish port forwarding and sync
5. Execute the specified command in the selected container (default: open a terminal)

```
Usage:
  devspace up [flags]

Flags:
  -b, --build                   Force image build
  -c, --container string        Container name where to open the shell
  -d, --deploy                  Force chart deployment
      --exit-after-deploy       Exits the command after building the images and deploying the devspace
  -h, --help                    help for up
      --init-registries         Initialize registries (and install internal one) (default true)
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
  -n, --namespace string        Namespace where to select pods
      --portforwarding          Enable port forwarding (default true)
      --switch-context          Switch kubectl context to the devspace context
      --sync                    Enable code synchronization (default true)
      --tiller                  Install/upgrade tiller (default true)
      --verbose-sync            When enabled the sync will log every file change

Examples:
devspace up                  # Start the devspace
devspace up bash             # Execute bash command after deploying
devspace up --switch-context # Change kubectl context to devspace context that is used
```
