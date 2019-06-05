---
title: Add Kubernetes manifests
---

DevSpace is able to deploy any kubernetes manifest via `kubectl apply -f`. Make sure you have `kubectl` installed for this to work.

## Deploy via kubectl

A minimal `devspace.yaml` config example can look like this:
```yaml
deployments:
- name: devspace-default
  kubectl:
    manifests:
    - kube
    - kube2
```

This will translate during deployment into the following commands:
```bash
kubectl apply -f kube
kubectl apply -f kube2
```

If you have an image defined in your `devspace.yaml` that should be build before deploying like this:
```yaml
images:
  default:
    # The name defined here is the name DevSpace will search for in kubernetes manifests
    image: dscr.io/yourusername/devspace
    createPullSecret: true
```

DevSpace will search through all the kubernetes manifests that should be deployed before actual deployment and replace any 
```yaml
image: dscr.io/yourusername/devspace
```

with 

```yaml
image: dscr.io/yourusername/devspace:the-tag-that-was-just-build
```

The replacement **only** takes place in memory and is **not** written to the filesystem and hence will **never** change any of your kubernetes manifests. This makes sure the just build image will actually be deployed.  

For a complete example using kubectl as deployment method take a look at [quickstart-kubectl](https://github.com/devspace-cloud/devspace/tree/master/examples/quickstart-kubectl)

## Kubectl configuration options

### deployments[\*].kubectl
```yaml
kubectl:                            # struct   | Options for deploying with "kubectl apply"
  cmdPath: ""                       # string   | Path to the kubectl binary (Default: "" = detect automatically)
  manifests: []                     # string[] | Array containing glob patterns for the Kubernetes manifests to deploy using "kubectl apply" (e.g. kube or manifests/service.yaml)
  kustomize: false                  # bool     | Use kustomize when deploying manifests via "kubectl apply" (Default: false)
  flags: []                         # string[] | Array of flags for the "kubectl apply" command
```
