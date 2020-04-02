---
title: "Command - devspace"
sidebar_label: devspace
---

## devspace

Welcome to the DevSpace!

### Synopsis

DevSpace accelerates developing, deploying and debugging applications with Docker and Kubernetes. Get started by running the init command in one of your projects:
	
		devspace init

### Options

```
      --config string         The devspace config file to use
      --debug                 Prints the stack trace if an error occurs
  -h, --help                  help for devspace
      --kube-context string   The kubernetes context to use
  -n, --namespace string      The kubernetes namespace to use
      --no-warn               If true does not show any warning when deploying into a different namespace or kube-context than before
  -p, --profile string        The devspace profile to use (if there is any)
      --silent                Run in silent mode and prevents any devspace log output except panics & fatals
  -s, --switch-context        Switches and uses the last kube context and namespace that was used to deploy the DevSpace project
      --var strings           Variables to override during execution (e.g. --var=MYVAR=MYVALUE)
```

```

```


## Flags
## Global & Inherited Flags