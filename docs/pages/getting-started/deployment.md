---
title: 2. Deploy with DevSpace
---

After installing DevSpace CLI, you are ready to deploy applications to Kubernetes with DevSpace CLI.

## Choose a project to deploy
You can either deploy one of your own projects or alternatively, checkout one of our demo applications using git:
<!--DOCUSAURUS_CODE_TABS-->
<!--Your Project-->
```bash
# Navigate to the root directory of your project
cd /path/to/your/project
```

<!--Node.js Demo-->
```bash
git clone https://github.com/devspace-cloud/quickstart-nodejs
cd quickstart-nodejs
```

<!--Python Demo-->
```bash
git clone https://github.com/devspace-cloud/quickstart-python
cd quickstart-python
```

<!--Golang Demo-->
```bash
git clone https://github.com/devspace-cloud/quickstart-golang
cd quickstart-golang
```

<!--Ruby Demo-->
```bash
git clone https://github.com/devspace-cloud/quickstart-ruby
cd quickstart-ruby
```

<!--PHP Demo-->
```json
git clone https://github.com/devspace-cloud/quickstart-php
cd quickstart-php
```
<!--END_DOCUSAURUS_CODE_TABS-->

## Initialize your project
Run this command in your project directory to initialize your application with DevSpace CLI:
```bash
devspace init
```

> If your project does not have a Dockerfile yet, DevSpace CLI will automatically create a Dockerfile for your project. Learn more about [containerizing your projects using DevSpace](/docs/workflow-basics/containerization).

While initializing your project, DevSpace CLI will ask you a couple of questions and then create a `devspace.yaml` file within your project which contains a basic configuration for building and deploying your application.


<details>
<summary>
### What is defined in the basic configuration of this `devspace.yaml`?
</summary>

```yaml
# Config version
version: v1beta2

# Development-specific configuration (will be explained later)
dev: ...

# Defines an array of everything (component, Helm chart, Kubernetes maninfests) 
# that will be deployed with DevSpace CLI in the specified order
deployments:
- name: quickstart-nodejs               # Name of this deployment
  component:                            # Deploy a component (alternatives: helm, kubectl)
    containers:                         # Defines an array of containers that run in the same pods started by this component
    - image: dscr.io/username/devspace  # Image of this container
      resources:
        limits:
          cpu: "400m"                   # CPU limit for this container
          memory: "500Mi"               # Memory/RAM limit for this container
    service:                            # Expose this component with a Kubernetes service
      ports:                            # Array of container ports to expose through the service
      - port: 3000                      # Exposes container port 3000 on service port 3000

# Defines all Dockerfiles that DevSpace CLI will build, tag and push
images:
  default:                              # Key 'default' = Name of this image
    image: dscr.io/username/devspace    # Registry and image name for pushing the image (dscr.io is the private registry provided by DevSpace Cloud)
    createPullSecret: true              # Let DevSpace CLI automatically create pull secrets in your Kubernetes namespace
```

</details>


## Create a Space (DevSpace Cloud)
*If you are **not** using DevSpace Cloud, you can skip this step.*

You can create an isolated Kubernetes namespace with the command:
```bash
devspace create space my-app
```
This command would create a Space called `my-app`. If you are using DevSpace Cloud with your own cluster (connected cluster), this namespace would be created within your own Kubernetes cluster but the Space would be isolated and managed by DevSpace Cloud.

> DevSpace CLI automatically uses the Space you just created for all following commands. Learn how to [switch between different Spaces](/docs/cloud/spaces/create-spaces#switch-between-spaces).


## Deploy your application
Now, you can deploy your application with the following command:
```bash
devspace deploy
```

This command will do the following:
1. Build the Dockerfiles specified in the `images` section of your `devspace.yaml`
2. Push the resulting Docker images to the specified registries
3. Create image pull secrets in your Kubernetes namespace 
4. Deploy everything that is defined unter `deployments` in your `devspace.yaml`

**Congrats you have successfully deployed an application to kubernetes!**

<details>
<summary>
### Learn more about image building with DevSpace
</summary>
DevSpace CLI builds and pushes your Docker images before deploying your projects. Follow these links to learn more about how to:
- [Configure image building](/docs/image-building/overview)
- [Add images to be built](/docs/image-building/add-images)
- [Authenticate with private Docker registries](/docs/image-building/registries/authentication)

DevSpace CLI will also create image pull secrets, if you configure this. Learn more about [image pull secrets](/docs/image-building/registries/pull-secrets).
</details>

<details>
<summary>
### Learn more about deploying with DevSpace
</summary>
DevSpace CLI lets you define the following types of deployments:
- Components ([What are components?](/docs/deployment/components/what-are-components))
- Helm charts ([What are Helm charts?](/docs/deployment/helm-charts/what-are-helm-charts))
- Kubernetes manifests ([What are Kubernetes manifests?](/docs/deployment/kubernetes-manifests/what-are-manifests))

<details>
<summary>
#### Deploy Components
</summary>
With DevSpace CLI, you can easily:
- [Add predefined components (e.g. a database) to your deployments](/docs/deployment/components/add-predefined-components)
- [Add custom components to your deployments](/docs/deployment/components/add-custom-components)

You can fully customize your components (predefined and custom) within your `devspace.yaml`:
- [Configure create and mount volumes](/docs/deployment/components/configuration/volumes)
- [Configure environment variables](/docs/deployment/components/containers/environment-variables)
- [Configure resource limits](/docs/deployment/components/containers/resource-limits)
- [Configure resource auto-scaling](/docs/deployment/components/configuration/scaling)
- [Expose components via services](/docs/deployment/components/configuration/service)
</details>

<details>
<summary>
#### Deploy Helm Charts
</summary>
If you want to deploy Helm charts, you can easily [add Helm charts to the deployment process](/docs/deployment/helm-charts/add-charts). This works for local Helm charts within your project or with Helm charts hosted on a registry.
</details>

<details>
<summary>
#### Deploy Kubernetes manifests
</summary>
If you want to deploy your existing Kubernetes manifests, you can easily [add these manifests to the deployment process](/docs/deployment/kubernetes-manifests/add-manifests).
</details>

</details>

## Open your app in the browser (DevSpace Cloud)
*If you are **not** using DevSpace Cloud, you will need to setup an ingress-controller, define an ingress and configure the DNS of your domain to point to your cluster in order to use `devspace open`.*

You can now view your application in the browser using the following command:
```bash
devspace open
```
If you are using DevSpace Cloud, your application will automatically be available on a `.devspace.host` subdomain. 

Learn how to [connect custom domains](/docs/cloud/spaces/domains). 

## What's next?
DevSpace CLI does more than simplify and streamline the process of deploying applications to Kubernetes. It also lets you:
- [Develop applications directly inside a Kubernetes cluster](/docs/getting-started/development)
- [Debug and analyze deployed applications](/docs/getting-started/debugging)
- [Example Configurations](https://github.com/devspace-cloud/devspace/tree/master/examples)
