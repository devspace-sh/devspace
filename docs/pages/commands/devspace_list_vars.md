---
title: "Command - devspace list vars"
sidebar_label: devspace list vars
---


Lists the vars in the active config

## Synopsis


```
devspace list vars [flags]
```

```
#######################################################
############### devspace list vars ####################
#######################################################
Lists the defined vars in the devspace config with their
values
#######################################################
```


## Flags

```
  -h, --help   help for vars
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

