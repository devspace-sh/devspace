---
title: Configure Manifest Deployments
sidebar_label: Manifests (kubectl)
---

To deploy plain Kubernetes manifests with `kubectl apply` or Kustomizations with `kustomize`, you need to configure them within the `deployments` section of the `devspace.yaml`.
```yaml
deployments:
- name: backend
  kubectl:
    manifests:
    - backend/
    - backend-extra/
- name: frontend
  kubectl:
    manifests:
    - frontend/manifest.yaml
```

The above example will be executing during the deployment process as follows:
```bash
kubectl apply -f backend
kubectl apply -f backend-extra
kubectl apply -f frontend/manifest.yaml
```

> Deployments with `kubectl` require `kubectl` to be installed. The `kubectl` binary either needs to be found through your `PATH` variable or by specifying the [`cmdPath` option](#cmdpath).

[What are Kubernetes manifests?](/docs/cli/deployment/kubernetes-manifests/what-are-manifests)


## Manifests

### `deployments[*].kubectl.manifests`
The `manifests` option expects an array of paths or path globs that point to Kubernetes manifest files (yaml or json files) or to folders containing Kubernetes manifests or Kustomizations.

> Configuring `manifests` is mandatory for `kubectl` deployments.

#### Example: Manifests
```yaml
deployments:
- name: backend
  kubectl:
    manifests:
    - backend/
    - backend-extra/
    - glob/path/to/manifests/*
```
**Explanation:**  
Instead of the default name `backend-headless`, the headless service for the ReplicaSet created by this component would be `custom-name-for-headless-service`.


### `deployments[*].kubectl.kustomize`
The `kustomize` option expects a boolean stating if DevSpace should deploy using `kustomize`.

> If you set `kustomize = true`, all of your `manifests` must be paths to Kustomizations. If you want to deploy some plain manifests and some Kustomizations, create multiple deployments for each of them.


#### Default Value for `kustomize`
```yaml
kustomize: false
```

#### Example: Kustomize
```yaml
deployments:
- name: backend
  kubectl:
    manifests:
    - kustomization1/
    - glob/path/to/more/kustomizations/*
    kustomize: true
```


### `deployments[*].kubectl.replaceImageTags`
The `replaceImageTags` option expects a boolean stating if DevSpace should do [Image Tag Replacement](/docs/cli/deployment/workflow-basics#3-tag-replacement).

By default, DevSpace searches all your manifests for images that are defined in the `images` section of the `devspace.yaml`. If DevSpace finds an image, it replaces or appends the image tag with the tag it created during the [image building process](/docs/cli/image-building/workflow-basics). Image tag replacement makes sure that your application will always be started with the most up-to-date image that DevSpace has built for you.

> Tag replacement takes place **in-memory** and is **not** writing anything to the filesystem, i.e. it will **never** change any of your configuration files.

#### Default Value for `replaceImageTags`
```yaml
replaceImageTags: true
```

#### Example: Disable Tag Replacement
```yaml
deployments:
- name: backend
  kubectl:
    manifests:
    - backend/
    - backend-extra/
    - glob/path/to/manifests/*
    replaceImageTags: false
```


## Kubectl Options

### `deployments[*].kubectl.flags`
The `flags` option expects an array of string stating additional flags and flag values that should be used when calling `kubectl apply`.

#### Default Value for `flags`
```yaml
flags: []
```

#### Example: Custom Kubectl Flags
```yaml
deployments:
- name: backend
  kubectl:
    manifests:
    - backend/
    flags:
    - --timeout
    - 10s
    - --grace-period
    - "30"
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
kubectl apply --timeout=10s --grace-period=30 -f backend/
```


### `deployments[*].kubectl.cmdPath`
The `cmdPath` option expects an array of string stating additional flags and flag values that should be used when calling `kubectl apply`.

> Setting `cmdPath` makes it much harder to share your `devspace.yaml` with other team mates. It is recommended to add `kubectl` to your `$PATH` environment variable instead.

#### Example: Setting Path To Kubectl Binary
```yaml
deployments:
- name: backend
  kubectl:
    manifests:
    - backend/
    cmdPath: /path/to/kubectl
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
/path/to/kubectl apply -f backend/
```



<br>

---
## Useful Commands

### `devspace add deployment [NAME] --manifests="./my/manifests/"`
```bash
devspace add deployment [deployment-name] --manifests="./path/to/your/manifests"
```
If you want to add existing Kubernetes manifests as deployments, you can do so by specifying a glob pattern for the `--manifests` flag as sown above. 

You can use [globtester.com](http://www.globtester.com/#p=eJzT0y9ILMnQL8nXr8wvLdLPTczLTEstLinW19ICAIcMCZc%3D&r=eJyVzMENgCAMAMBVDAPQBSq7VKiRhAKhlYTt9e3PAe4w5bnFQqq7E7J4ueChk11gDVa7BwjVfLKaQuJe2hKu5hdJwWMEhNcH%2FJEoj5kjf4YH8%2BAw7w%3D%3D&) to verify that your pattern matches the relative paths to your manifests. Paths should be relative to the root directory of your project which also contains your `devspace.yaml`.

### `devspace remove deployment [NAME]`
Instead of manually removing a deployment from your `devspace.yaml`, it is recommended to run this command instead:
```bash
devspace remove deployment [deployment-name]
```

The benefit of running `devspace remove deployment` is that DevSpace will ask you this question:
```bash
? Do you want to delete all deployment resources deployed?  [Use arrows to move, type to filter]
> yes
  no
```

If you select yes, DevSpace  will remove your deployment from your Kubernetes cluster before deleting it in your `devspace.yaml`. This is great to keep your Kubernetes namespaces clean from zombie deployments that cannot be easily tracked, removed and updated anymore.
