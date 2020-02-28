---
title: "Command - devspace add"
sidebar_label: devspace add
---


Convenience command: adds something to devspace.yaml

## Synopsis


```
#######################################################
#################### devspace add #####################
#######################################################
Adds config sections to devspace.yaml
```
## Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

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

## See Also
* [devspace add deployment](devspace_add_deployment.md)	 - Adds a deployment to devspace.yaml
* [devspace add image](devspace_add_image.md)	 - Add an image
* [devspace add port](devspace_add_port.md)	 - Add a new port forward configuration
* [devspace add provider](devspace_add_provider.md)	 - Adds a new cloud provider to the configuration
* [devspace add sync](devspace_add_sync.md)	 - Add a sync path
