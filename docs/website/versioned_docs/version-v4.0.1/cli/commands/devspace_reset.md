---
title: Command - devspace reset
sidebar_label: devspace reset
id: version-v4.0.1-devspace_reset
original_id: devspace_reset
---


Resets an cluster token

## Synopsis


```
#######################################################
################## devspace reset #####################
#######################################################
```
## Options

```
  -h, --help   help for reset
```

### Options inherited from parent commands

```
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
* [devspace reset key](../../cli/commands/devspace_reset_key)	 - Resets a cluster key
* [devspace reset vars](../../cli/commands/devspace_reset_vars)	 - Resets the current config vars
