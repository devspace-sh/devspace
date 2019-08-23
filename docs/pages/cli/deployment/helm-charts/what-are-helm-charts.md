---
title: What are Kubernetes Helm Chart Deployments?
sidebar_label: Helm Charts
---

[Helm](https://helm.sh/) is the package manager for Kubernetes. Packages in Helm are called Helm charts.

> If you just want to add a kubernetes yaml to the chart take a look at [add custom kubernetes files](/docs/deployment/kubernetes-manifests/add-manifests)

## Structure of an Helm chart
The following structure shows the most important parts of an Helm chart:
```bash
chart/              # Helm chart (defines how to deploy your application)
|-Chart.yaml        # chart definition (e.g. name, version)
|-requirements.yaml # OPTIONAL: dependencies (other charts) that will be deployed together with your chart
|-values.yaml       # values for the template variables
|-templates/        # directory containing the template files (Kubernetes manifests)
```
### Chart.yaml
The `Chart.yaml` describes basic information about your Chart, e.g. the name, description or version of your chart.
```yaml
name: my-app
version: v0.0.1
description: A Kubernetes-Native Application
```

### values.yaml
The `values.yaml` defines the default values that are used for parsing the templates defined in `templates/` when deploying the Helm chart.

The following code snippet shows how an exemplary `values.yaml` could look like:
```yaml
replicas: 3
containers:
- name: container-1
  image: my-registry.tld/my-image:tag
  env:
  - name: MY_ENV
    value: "some env var value"
  resources:
    limits:
      cpu: "200m"
      memory: "300Mi"
```
The values defined in `values.yaml` are defaults which can be overridden during the deployment of an Helm chart. DevSpace CLI uses value-overriding to update the image tags to the most recently build and pushed tags.

### templates/
The `templates/` folder contains all templates for your chart. Tiller will parse all the `.yaml` files defined in this folder and parse them as templates together with the values defined in the `values.yaml`. 

The following code snippet shows a simplified version of a template file within a Helm chart:
```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: {{ .Values.replicas | default 1 }}
  selector: ...
  template:
    metadata: ...
    spec:
      containers:
        {{- range $containerIndex, $container := Values.containers }}
        - name: {{ $container.name | default "container" | quote }}
          image: {{ $container.image | quote }}
          env:
{{ toYaml $container.env | indent 12 }}
          {{- if $container.resources }}
          resources:
            {{- with $container.resources.limits }}
            limits:
              cpu: {{ .cpu | quote }}
              memory: {{ .memory | quote }}
            {{- end }}
          {{- end }}
        {{- end }}
```
[Learn more about templating in Helm](https://docs.helm.sh/chart_template_guide/).

### requirements.yaml
The `requirements.yaml` defines the dependencies of your Helm chart. If your application needs a mysql database for example, you could add the mysql chart as dependency. Defining such a dependency in the `requirements.yaml` could look like this:
```yaml
dependencies:
- name: mysql
  version: 3.2.1
```
DevSpace CLI provides the convenience command `devspace add package [chart-name]` to add dependencies to your Helm chart. This command will not only add a dependency to your chart but also add the most important values of this chart to your `values.yaml` and show you the `README` of the newly added chart, so you can easily customize the dependency.

[Learn more about adding predefined components.](/docs/deployment/components/add-predefined-components)

---
## FAQ

<details>
<summary>
### Do I need to install Helm to use DevSpace CLI?
</summary>
**No.** DevSpace CLI comes with an in-built Helm client and it will automatically install [Tiller](#what-is-tiller), the server-side Helm component, within your Spaces.
</details>

<details>
<summary>
### How does DevSpace CLI deploy charts?
</summary>
When you run `devspace deploy` or `devspace dev`, DevSpace CLI will deploy your chart. This deployment process involves the following steps:
1. Installing or upgrading [Tiller](#what-is-tiller) in your Space
2. Loading the template values from `values.yaml`
3. Updating the image tags in the template values to the most recently image that has been built and pushed by DevSpace CLI (happens in-memory)
4. Opening a connection to the [Tiller](#what-is-tiller) server in your Space (via port-forwarding)
5. Deploying the chart with [Tiller](#what-is-tiller) as new release OR upgrading an existing release
6. [ON ERROR: rollback release to the latest working version (revision)]
</details>

<details>
<summary>
### How do I update a deployed Helm chart with DevSpace?
</summary>
If you changed your chart (e.g. edited the values.yaml), you can simply run `devspace deploy` again and DevSpace CLI will update your existing Helm release (i.e. deployed application).
</details>

<details>
<summary>
### Should I add an ingress template to `templates/`?
</summary>
Generally: **No.** 

The problem with adding an ingress to your Helm chart is that you cannot share your code with other developers because the same hostname (domain) can only be used by one person, otherwise there would be two ingresses using the same domain which will cause problems with the Kubernetes-internal traffic routing. 

You can, however, manually create ingresses or manually edit any ingress that has been automatically created. Use the following command to edit an ingress manually:
```bash
kubectl edit ingress [INGRESS_NAME]
```
Use `kubectl get ingress` to list all ingresses in a Space.
</details>

<details>
<summary>
### What is Tiller?
</summary>
Tiller is the server-side component of Helm which is responsible for instantiating releases within your Kubernetes namespace and for keeping track of different revisions of a release that you deploy over time. Tiller will likely be removed in the future because a lot of Helm users want Helm to become a client-only tool.

Before deploying your application, DevSpace CLI will start a Tiller deployment within your Space which then deploys your application as defined in your Helm chart. You can actually see the tiller pod by running this kubectl command:
```bash
kubectl get po -l name=tiller 
```
</details>

<details>
<summary>
### Can I use DevSpace without Helm?
</summary>
**Yes**. You can simply [add plain Kubernetes manifests as deployments](/docs/deployment/kubernetes-manifests/add-manifests) and DevSpace CLI will run `kubectl apply -f [FILE]` instead of using Helm.
</details>
