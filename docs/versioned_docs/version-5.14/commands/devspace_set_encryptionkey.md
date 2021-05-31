---
title: "Command - devspace set encryptionkey"
sidebar_label: devspace set encryptionkey
---


Sets the encryption

## Synopsis


```
devspace set encryptionkey [flags]
```

```
#######################################################
############## devspace set encryptionkey #############
#######################################################
Sets an encryption key for a given cluster

Examples:
devspace set encryptionkey mykey --cluster mycluster 
devspace set encryptionkey --cluster mycluster --reset
#######################################################
```


## Flags

```
      --cluster string   The cluster to apply this key for
  -h, --help             help for encryptionkey
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

