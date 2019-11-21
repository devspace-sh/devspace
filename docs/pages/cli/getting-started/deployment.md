---
title: 2. Deploy to Kubernetes
---

After installing DevSpace, you are ready to deploy your first project.

## Choose a Project
You can either deploy one of your own projects or alternatively, checkout one of our demo applications using git:
<!--DOCUSAURUS_CODE_TABS-->
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

<!--PHP Demo-->
```json
git clone https://github.com/devspace-cloud/quickstart-php
cd quickstart-php
```

<!--Ruby Demo-->
```bash
git clone https://github.com/devspace-cloud/quickstart-ruby
cd quickstart-ruby
```

<!--Your Project-->
```bash
# Navigate to the root directory of your project
cd /path/to/your/project
```

<!--END_DOCUSAURUS_CODE_TABS-->

> If you are using DevSpace for the first time, it is highly recommended that you use one of the demo projects.

## Initialize Your Project
Run this command in your project directory to create a `devspace.yaml` config file for your project:
```bash
devspace init
```

While initializing your project, DevSpace will ask you a couple of questions and then create the config file `devspace.yaml` which will look similar to this one:

```yaml
# Config version
version: v1beta5

# Defines all Dockerfiles that DevSpace will build, tag and push
images:
  default:                              # Key 'default' = Name of this image
    image: reg.tld/username/devspace    # Registry and image name for pushing the image
    createPullSecret: true              # Let DevSpace automatically create pull secrets in your Kubernetes namespace

# Defines an array of everything (component, Helm chart, Kubernetes maninfests) 
# that will be deployed with DevSpace in the specified order
deployments:
- name: quickstart-nodejs                 # Name of this deployment
  helm:                                   # Deploy using Helm
    componentChart: true                  # Use the Component Helm Chart
    values:                               # Override Values for chart (van also be set using valuesFiles option)
      containers:                         # Defines an array of containers that run in the same pods started by this component
      - image: reg.tld/username/devspace  # Image of this container
      service:                            # Expose this component with a Kubernetes service
        ports:                            # Array of container ports to expose through the service
        - port: 3000                      # Exposes container port 3000 on service port 3000

# Settings for development mode (will be explained later)
dev: ...
```

> If you already have Kubernetes manifests or a Helm chart, tell DevSpace during the init process about these resources and you will see a [kubectl deployment](../../cli/deployment/kubernetes-manifests/configuration/overview-specification) or a [helm deployment](../../cli/deployment/helm-charts/configuration/overview-specification) in the `devspace.yaml` instead of the [component deployment](../../cli/deployment/components/configuration/overview-specification) shown in this config snippet.


## Choose a Kubernetes Cluster
Choose the cluster, you want to deploy your project to. If you are not sure, pick the first option. It is very easy to switch between the options later on.
<br>

<details>
<summary><h3 style="margin-bottom: 0;">Hosted Spaces sponsored by DevSpace (managed k8s namespaces)</h3>
<i>
<br>&nbsp;&nbsp;&nbsp;&nbsp;
FREE for one project, includes 1 GB RAM
</i>
</summary>

At DevSpace, we believe everybody should have access to Kubernetes. That's why we sponsor free Kubernetes namespaces with 1GB RAM for everyone. You can simply create such a Space using this command:

```bash
devspace create space my-app # requires login via GitHub or email
```
> DevSpace automatically sets up a kube-context for this space, so you can also access your isolated namespace using `kubectl`, `helm` or any other Kubernetes tool.

</details>

<details>
<summary><h3 style="margin-bottom: 0;">Your own local cluster</h3>
<i>
<br>&nbsp;&nbsp;&nbsp;&nbsp;
works with any local Kubernetes cluster (minikube, kind, k3s, mikrok8s etc.)
</i>
</summary>

If you want to deploy to a local Kubernetes cluster, make sure your **current kube-context** points to this cluster and tell DevSpace which namespace to use:

```bash
# Tell DevSpace which namespace to use (will be created automatically during deployment)
devspace use namespace my-namespace
```

</details>

<details>
<summary><h3 style="margin-bottom: 0;">Your own remote cluster</h3>
<i>
<br>&nbsp;&nbsp;&nbsp;&nbsp;
works with any remote Kubernetes cluster (GKE, EKS, AKS, bare metal etc.)
</i>
</summary>


<details>
<summary><h4>Option A: You want to use this cluster alone</h4></summary>

If you want to deploy to a remote Kubernetes cluster, make sure your **current kube-context** points to this cluster and tell DevSpace which namespace to use:

```bash
# Tell DevSpace which namespace to use (will be created automatically during deployment)
devspace use namespace my-namespace
```

</details>

<details>
<summary><h4>Option B: You want to share this cluster with your team</h4></summary>

To share a cluster, connect it to [DevSpace Cloud](../../cloud/what-is-devspace-cloud) and then create an isolated Kubernetes namespace.

```bash
# Connect your cluster to DevSpace Cloud
devspace connect cluster # requires login via GitHub or email

# Create an isolated Kubernetes namespace in your cluster via DevSpace Cloud
devspace create space my-namespace
```

> DevSpace automatically sets up a kube-context for every space you create, so you can also access your isolated namespace using `kubectl`, `helm` or any other Kubernetes tool.

<details>
  <summary><h5>What is DevSpace Cloud?</h5></summary>

[DevSpace Cloud](../../cloud/what-is-devspace-cloud) is the optional server-side component for DevSpace that allows you to connect any Kubernetes cluster and then share it with your team for development. DevSpace Cloud lets developers create isolated Kubernetes namespaces on-demand and makes sure that developers cannot break out of their namespaces by configuring RBAC, network & pod security policies etc.

> You can either
> - use the fully managed **[SaaS edition of DevSpace Cloud](https://app.devspace.cloud)**
> - or run it on your clusters using the <img width="20px" style="vertical-align: sub" src="/img/logos/github-logo.svg" alt="DevSpace Demo"> **[on-premise edition available on GitHub](https://github.com/devspace-cloud/devspace-cloud)**.

</details>

<details>
  <summary><h5>How are Spaces isolated? Why is it safe to share a cluster?</h5></summary>

DevSpace Cloud makes sure that developers cannot break out of their namespaces by configuring RBAC, network policies, pod security policies etc. By default, these restrictions are very strict and do not even allow pods from different namespaces to communicate with eather other. You can configure every security setting that DevSpace Cloud enforces using the UI of DevSpace Cloud and even set custom limits for different members of your team.

</details>

<details>
  <summary><h5>How can I add my team mates, so we can share this cluster?</h5></summary>

1. Connect your cluster to DevSpace Cloud using `devspace connect cluster`
2. Go to **Clusters** in the UI of DevSpace Cloud: [https://app.devspace.cloud/clusters](https://app.devspace.cloud/clusters)
3. Click on your cluster
4. Go to the **Invites** tab
5. Click on the **Add Invite** button
6. Click on the invite link in the table and send the link to a team mate
7. After clicking on the link and defining an encryption key, your team mate will be able to create isolated namespaces.

</details>

<details>
  <summary><h5>It it safe to connect my cluster to DevSpace Cloud?</h5></summary>

**Yes**. When connecting a cluster to DevSpace Cloud, the CLI tool asks you to define an encrytion key. The cluster access token that the CLI creates will be encrypted with a hashed version of this key before sending it to DevSpace Cloud. That makes sure that no one can access your cluster except you. This key is hashed and stored on your local computer. That means that:

- If you use DevSpace from a different computer, you will have to enter the encryption key again or re-connect the cluster which generates a new access token and encrypts it with a new key.
- If you add a team member, you will have to send them a secure invite link which makes sure that they also get cluster access. This procedure is very safe and your key is never sent to our platform. After clicking on the invite link, your colleagues will define a separate encryption key for secure access to their namespaces.

> If you are still hesitant, you can also run DevSpace Cloud yourself in your own Kubernetes cluster using the <img width="20px" style="vertical-align: sub" src="/img/logos/github-logo.svg" alt="DevSpace Demo"> **[on-premise edition available on GitHub](https://github.com/devspace-cloud/devspace-cloud)**

</details>

<details>
  <summary><h5>Can I run DevSpace Cloud on-premise in my own cluster?</h5></summary>
  
  **Yes**. Follow these intructions to run DevSpace Cloud yourself:

  **1. Install DevSpace Cloud**  
  &nbsp;&nbsp;&nbsp;
  Follow the **[install instructions for DevSpace Cloud on-premise](https://github.com/devspace-cloud/devspace-cloud)** available on <img width="20px" style="vertical-align: sub" src="/img/logos/github-logo.svg" alt="DevSpace Demo"> **GitHub**.

  **2. Tell DevSpace to use your self-hosted DevSpace Cloud**  
```bash
devspace use provider devspace.my-domain.tld
```

  **3. Connect a Kubernetes cluster to your self-hosted DevSpace Cloud**  
```bash
devspace connect cluster
```

  **4. Create an isolated namespace**  
```bash
devspace create space my-app
```

  </details>

</details>

</details>


## Deploy Your Application
Now, you can deploy your application with the following command:
```bash
devspace deploy
```

This command will do the following:
1. Build the Dockerfile(s) specified in the `images` section of your `devspace.yaml`
2. Tag the resulting image(s) with an auto-generated tag according to a [customizable tag schema](../../cli/image-building/configuration/overview-specification#images-tag-tagging-schema)
3. Push the resulting Docker images to the specified registries
4. Create image pull secrets in your Kubernetes namespace (optional)
5. Deploy everything that is defined unter `deployments` in your `devspace.yaml`

<img src="/img/processes/deployment-process-devspace.svg" alt="DevSpace Deployment Process" style="width: 100%;">


## Open Your Application
You can now open your application in the browser using the following command:
```bash
devspace open
```
When DevSpace asks you how to open your application, choose the first option: **via localhost**
```bash
? How do you want to open your application?
  [Use arrows to move, space to select, type to filter]
> via localhost (provides private access only on your computer via port-forwarding) # <<<<<<<< CHOOSE THIS ONE!
  via domain (makes your application publicly available via ingress)
```
To use the second option, you either need to make sure the DNS of your domain points to your Kubernetes cluster and you have an ingress-controller running in your cluster OR you use [DevSpace Cloud](../../cloud/what-is-devspace-cloud), either in form of Hosted Spaces or by connecting your own cluster using the command `devspace connect cluster`.

> **Congratulations!** You just deployed your first project to Kubernetes using DevSpace.

<img style="float: left; max-width: 500px; margin-right: 50px;" src="/img/congrats.gif">

## What's next?
DevSpace ist not just a deployment tool, it is also a very powerful development tool. And that is actually the most powerful part of DevSpace. So, don't stop now.

In the last step of this Getting Started Guide, you will learn how to [develop applications directly inside a Kubernetes cluster](../../cli/getting-started/development).
