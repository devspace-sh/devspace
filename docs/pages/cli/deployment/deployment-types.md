---
title: Types of Deployments
---

DevSpace is able to deploy any kubernetes manifest via `kubectl apply -f`. Make sure you have `kubectl` installed for this to work.

> For a complete example using kubectl as deployment method take a look at [quickstart-kubectl](https://github.com/devspace-cloud/devspace/tree/master/examples/quickstart-kubectl)

## Deploy via kubectl

A minimal `devspace.yaml` deployment config example can look like this:
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


## Kubectl deployment configuration options

### deployments[\*].kubectl
```yaml
kubectl:                            # struct   | Options for deploying with "kubectl apply"
  cmdPath: ""                       # string   | Path to the kubectl binary (Default: "" = detect automatically)
  manifests: []                     # string[] | Array containing glob patterns for the Kubernetes manifests to deploy using "kubectl apply" (e.g. kube or manifests/service.yaml)
  kustomize: false                  # bool     | Use kustomize when deploying manifests via "kubectl apply" (Default: false)
  flags: []                         # string[] | Array of flags for the "kubectl apply" command
```







DevSpace CLI lets you deploy one or even multiple applications. In DevSpace a **deployment** defines a specific part of an application or a whole application that should be deployed.  The configuration for these deployments can be found in the `deployments` section within your `devspace.yaml`.

## Deployment process
Running `devspace deploy` or `devspace dev` will do the following:
1. Build all Docker [`images` that you specified in `devspace.yaml`](/docs/image-building/configuration)
2. Push the Docker images to any [Docker registry](/docs/image-building/registries/authentication)
3. Create [image pull secrets](/docs/image-building/registries/pull-secrets) if specified
4. Deploy all deployments defined in the `devspace.yaml` in the specified order with the built images

## Types of deployments
DevSpace CLI lets you define the following types of deployments:
- [Components (Easy way to deploy common kubernetes resources)](/docs/deployment/components/what-are-components)
- [Helm charts](/docs/deployment/helm-charts/what-are-helm-charts)
- [Kubernetes manifests](/docs/deployment/kubernetes-manifests/what-are-manifests)
- [Kustomize manifests](/docs/deployment/kubernetes-manifests/kustomize)

> Take a look at the [examples](https://github.com/devspace-cloud/devspace/tree/master/examples) if you want to see example configurations for a certain deployment method.

## Structure of a deployment
A standard `devspace.yaml` with a single deployment could look like this:
```yaml
# An array of deployments (kubectl, helm, component) which will be deployed with DevSpace CLI in the specified order
deployments:
- name: my-deployment                   # Name of this deployment
  # Choose ONE of the following three deployment methods
  kubectl: ...                          # Deploy kubernetes or kustomize manifests (kubectl has to be installed locally)
  helm: ...                             # Deploy a local or remote helm chart
  component:                            # Deploy a component (alternatives: helm, kubectl)
    containers:                         # Defines an array of containers that run in the same pods started by this component
    - image: dscr.io/username/devspace  # Image of this container
    service:                            # Expose this component with a Kubernetes service
      ports:                            # Array of container ports to expose through the service
      - port: 3000                      # Exposes container port 3000 on service port 3000
```
This deployment is configured to deploy the [Helm chart for DevSpace Components](/docs/deployment/components/what-are-components) using the values specified in the `component` section.

Unlike `images` in the `devspace.yaml`, the `deployments` section is an array and not a key-value map because DevSpace CLI will iterate over the deployment one after another in the specified order and deploy it. This is useful because the order in which your deployments are starting might be relevant depending on your application.

## Add additonal deployments
DevSpace provides convenience commands for adding deployments to the `devspace.yaml`. If you don't want to add additional deployments in the config manually, you have the following options:

<details>
<summary>
### Add prefined components (e.g. a database)
</summary>
Run the following command to add a predefined component to your deployments:
```bash
devspace add deployment [deployment-name] --component=[component-name]
```
Example: `devspace add deployment database --component=mysql`

#### List of predefined components
DevSpace CLI provides the following predefined components:
- mariadb
- mongodb
- mysql
- postgres
- redis
</details>


<details>
<summary>
### Add custom components for existing Dockerfiles
</summary>
Run one of the following commands to add a custom component to your deployments based on an existing Dockerfile:
```bash
devspace add deployment [deployment-name] --dockerfile=""
devspace add deployment [deployment-name] --dockerfile="" --image="my-registry.tld/[username]/[image]"
```
The difference between the first command and the second one is that the second one specifically defines where the Docker image should be pushed to after building the Dockerfile. In the first command, DevSpace CLI would assume that you want to use the [DevSpace Container Registry](/docs/cloud/images/dscr-io) provided by DevSpace Cloud.

> If you are using a private Docker registry, make sure to [login to this registry](/docs/image-building/registries/authentication).

</details>

<details>
<summary>
### Add custom components for existing images
</summary>
If you want to use a Docker image from Docker Hub or any other registry, you can add a custom component to your deployments using this command:
```bash
devspace add deployment [deployment-name] --image="my-registry.tld/my-username/image"
```
Example using Docker Hub: `devspace add deployment database --image="mysql"`

> If you are using a private Docker registry, make sure to [login to this registry](/docs/image-building/registries/authentication).

</details>

<details>
<summary>
### Add existing Kubernetes manifests
</summary>
```bash
devspace add deployment [deployment-name] --manifests="./path/to/your/manifests"
```
If you want to add existing Kubernetes manifests as deployments, you can do so by specifying a glob pattern for the `--manifests` flag as sown above. 

You can use [globtester.com](http://www.globtester.com/#p=eJzT0y9ILMnQL8nXr8wvLdLPTczLTEstLinW19ICAIcMCZc%3D&r=eJyVzMENgCAMAMBVDAPQBSq7VKiRhAKhlYTt9e3PAe4w5bnFQqq7E7J4ueChk11gDVa7BwjVfLKaQuJe2hKu5hdJwWMEhNcH%2FJEoj5kjf4YH8%2BAw7w%3D%3D&) to verify that your pattern matches the relative paths to your manifests. Paths should be relative to the root directory of your project which also contains your `devspace.yaml`.
</details>

<details>
<summary>
### Add Helm charts (local and from registries)
</summary>

#### Add a local Helm chart
If you built your own Helm chart and it is located inside your project directory, you can simply add it as a deployment using the following command:
```bash
devspace add deployment [deployment-name] --chart="./path/to/my/chart"
```

#### Add a Helm chart from a Helm repository
If you want to deploy a Helm chart from a chart repository, you can simply add it as shown in this example:
```bash
devspace add deployment [deployment-name] --chart="stable/mysql"
```
You can replace `stable` with the name of your Helm chart repository, if it already exists on your local computer. If you want to use a chart from a chart repository that you have not used yet, you can also specify the repository URL:
```bash
devspace add deployment [deployment-name] --chart="chart-name" --chart-repo="https://my-chart-repository.tld"
```
> Use the `--chart-version` flag to specifiy the char version that you want to deploy.
</details>

After adding a new deployment, you need to manually redeploy in order to start the newly added component together with the remainder of your previouly existing deployments.
```bash
devspace deploy
```


## Removing deployments
If you want to delete a deployment from kubernetes you can run:
```bash
# Removes all deployments remotely
devspace purge
# Removes deployment with given name
devspace purge --deployments=my-deployment-1,my-deployment-2
```

## Removing deployments from the config
If you want to remove the deployment from the configuration, DevSpace CLI provides a convenient command `devspace remove deployment`, so instead of manually removing a deployment from your `devspace.yaml`, it is recommended to run this command instead:
```bash
devspace remove deployment [deployment-name]
```

The benefit of running `devspace remove deployment` is that DevSpace CLI will ask you this question:
```bash
? Do you want to delete all deployment resources deployed?  [Use arrows to move, type to filter]
> yes
  no
```

If you select yes, DevSpace CLI will remove your deployment from your Kubernetes cluster before deleting it in your `devspace.yaml`. This is great to keep your Kubernetes namespaces clean from zombie deployments that cannot be easily tracked, removed and updated anymore.
