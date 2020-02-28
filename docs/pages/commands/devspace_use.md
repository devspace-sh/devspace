---
title: "Command - devspace use"
sidebar_label: devspace use
---


Use specific config

## Synopsis


```
#######################################################
#################### devspace use #####################
#######################################################
```
## Options

```
  -h, --help   help for use
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
* [devspace use context](devspace_use_context.md)	 - Tells DevSpace which kube context to use
* [devspace use namespace](devspace_use_namespace.md)	 - Tells DevSpace which namespace to use
* [devspace use profile](devspace_use_profile.md)	 - Use a specific DevSpace profile
* [devspace use provider](devspace_use_provider.md)	 - Change the default provider
* [devspace use space](devspace_use_space.md)	 - Use an existing space for the current configuration
