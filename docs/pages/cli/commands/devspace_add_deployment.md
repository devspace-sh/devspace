---
title: "Command - devspace add deployment"
sidebar_label: deployment
---


Add a deployment

## Synopsis

 
```
devspace add deployment [deployment-name] [flags]
```

```
#######################################################
############# devspace add deployment #################
#######################################################
Add a new deployment (docker image, components, 
kubernetes manifests or helm chart) to your DevSpace configuration

Examples:
# Deploy a predefined component 
devspace add deployment my-deployment --component=mysql
# Deploy a local dockerfile
devspace add deployment my-deployment --dockerfile=./Dockerfile
devspace add deployment my-deployment --image=myregistry.io/myuser/myrepo --dockerfile=frontend/Dockerfile --context=frontend/Dockerfile
# Deploy an existing docker image
devspace add deployment my-deployment --image=mysql
devspace add deployment my-deployment --image=myregistry.io/myusername/mysql
# Deploy local or remote helm charts
devspace add deployment my-deployment --chart=chart/
devspace add deployment my-deployment --chart=stable/mysql
# Deploy local kubernetes yamls
devspace add deployment my-deployment --manifests=kube/pod.yaml
devspace add deployment my-deployment --manifests=kube/* --namespace=devspace
#######################################################
```
## Options

```
      --chart string                                   A helm chart to deploy (e.g. ./chart or stable/mysql)
      --chart-repo string                              The helm chart repository url to use
      --chart-version string                           The helm chart version to use
      --component devspace list available-components   A predefined component to use (run devspace list available-components to see all available components)
      --context string                                 
      --dockerfile string                              A dockerfile
  -h, --help                                           help for deployment
      --image string                                   A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)
      --manifests string                               The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)
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

* [devspace add](../../cli/commands/devspace_add)	 - Change the DevSpace configuration
