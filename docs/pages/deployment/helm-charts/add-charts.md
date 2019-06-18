---
title: Add Helm charts
---

DevSpace CLI lets you deploy existing Helm charts (either from your local filesystem or from a Helm registry).

> For a complete example using helm as deployment method take a look at [minikube](https://github.com/devspace-cloud/devspace/tree/master/examples/minikube)

## Deploy via helm

A minimal `devspace.yaml` deployment config example can look like this:
```yaml
deployments:
- name: default
  helm:
    chart:
      name: ./chart
```

This tells DevSpace to deploy a local chart in `./chart`. If you want to deploy a remote chart you can also specify:
```yaml
deployments:
- name: default
  helm:
    chart:
      name: redis
      version: "6.1.4"
      repo: https://kubernetes-charts.storage.googleapis.com
```

If you have an image defined in your `devspace.yaml` that should be build before deploying like this:
```yaml
images:
  default:
    # The name defined here is the name DevSpace will search for in kubernetes manifests
    image: dscr.io/yourusername/devspace
    createPullSecret: true
```

DevSpace will search through all the override values defined in the local chart at `localchartpath/values.yaml` or defined in `deployments[].helm.values` or `deployments[].helm.valuesFiles` and replace the image name `dscr.io/yourusername/devspace` with the image name and the just build tag.  

The replacement **only** takes place in memory and is **not** written to the filesystem and hence will **never** change any of your configuration files. This makes sure the just build image will actually be deployed.  

## Helm deployment configuration options

### deployments[\*].helm
```yaml
helm:                               # struct   | Options for deploying with Helm
  chart: ...                        # struct   | Relative path 
  wait: false                       # bool     | Wait for pods to start after deployment (Default: false)
  rollback: false                   # bool     | Rollback if deployment failed (Default: false)
  force: false                      # bool     | Force deleting and re-creating Kubernetes resources during deployment (Default: false)
  timeout: 180                      # int      | Timeout to wait for pods to start after deployment (Default: 180)
  tillerNamespace: ""               # string   | Kubernetes namespace to run Tiller in (Default: "" = same a deployment namespace)
  devSpaceValues: true              # bool     | If DevSpace CLI should replace images overrides and values.yaml before deploying (Default: true)
  valuesFiles:                      # string[] | Array of paths to values files
  - ./chart/my-values.yaml          # string   | Path to a file to override values.yaml with
  values: {}                        # struct   | Any object with Helm values to override values.yaml during deployment
```

### deployments[\*].helm.chart
```yaml
chart:                              # struct   | Chart to deploy
  name: my-chart                    # string   | Chart name
  version: v1.0.1                   # string   | Chart version
  repo: "https://my-repo.tld/"      # string   | Helm chart repository
  username: "my-username"           # string   | Username for Helm chart repository
  password: "my-password"           # string   | Password for Helm chart repository
```
