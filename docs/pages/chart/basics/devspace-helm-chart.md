---
title: DevSpace Helm Chart
---

Running `devspace init` will automatically add the DevSpace Helm Chart to the folder `chart/` within your project. The chart contains the information how to deploy the application to a kubernetes cluster. This chart is highly customizable and provides very powerful features out-of-the-box (e.g. horizontal auto-scaling).

## Configure the DevSpace Helm Chart

The `chart/values.yaml` is the most important place for configuring your Helm chart. See the following guides to learn how to use the `chart/values.yaml` to:
- [Configure persistent volumes](/docs/chart/customization/persistent-volumes)
- [Configure environment variables](/docs/chart/customization/environment-variables)
- [Configure networking (e.g. define services)](/docs/chart/customization/networking)
- [Add a predefined component (e.g. database)](/docs/chart/customization/predefined-components)
- [Add a custom component](/docs/chart/customization/add-component)
- [Add a sidecar container](/docs/chart/customization/containers)
- [Add custom Kubernetes files](/docs/chart/customization/custom-manifests)
- [Add helm packages](/docs/chart/customization/packages)

<details>
<summary>
#### Edit the `chart/Chart.yaml`
</summary>
It is recommended to change the `name` and `description` of your chart by editing the `Chart.yaml` and to update the `version` whenever you edit anything within your chart as described below.

[Learn more about versioning your chart](https://docs.helm.sh/chart_best_practices/#version-numbers)

</details>

<details>
<summary>
#### Show an example of the `values.yaml`
</summary>
```yaml
components:
- name: default
  replicas: 1
  containers:
  - name: default
    image: dscr.io/username/image
    command:
    - "sleep"
    args:
    - "999999999"
    resources:
      limits:
        cpu: "200m"
        memory: "300Mi"
        ephemeralStorage: "1Gi"
      requests: 
        cpu: "100m"
        memory: "100Mi"
        ephemeralStorage: "500Mi"
    env:
    - name: MY_ENV_VAR
      value: "test123"
    volumeMounts:
    - containerPath: /usr/share/nginx/html
      volume:
        name: nginx
        path: /nginx/html
        readOnly: false
  service:
    name: external
    type: ClusterIP
    ports:
    - externalPort: 80
      containerPort: 3000
  autoScaling:
    horizontal:
      maxReplicas: 4
      averageCPU: 80
      averageMemory: "200Mi"

volumes:
- name: nginx
  size: "1Gi"

pullSecrets:
- custom-pull-secret
```
</details>

By default, `devspace init` will create a minimal `values.yaml` containing the most important configuration options.

### Add dependencies in `requirements.yaml`
Generally, it is recommended to use `devspace add package [CHART_NAME]` to add a dependency and `devspace remove package [CHART_NAME]` to remove a dependency instead of manually editing the `requirements.yaml`. However, it can be useful to edit the `requirements.yaml` to change the version of a dependency.

Learn more about [adding and removing packages](/docs/chart/customization/packges).

### Customize `templates/`

> It is highly recommended **NOT** to edit any files within the `template/` folder of the DevSpace Helm Chart.

You can [add custom templates or Kubernetes manifests](/docs/chart/customization/custom-manifests) if needed. It is, however, recommended that you store them instide `templates/custom/` to allow you to run `devspace update chart` to [update the DevSpace Helm Chart](#update-the-devspace-helm-chart) without breaking anything.

## Update the DevSpace Helm Chart
The DevSpace Helm Chart is constantly being improved. To get the newest version of it, you can run `devspace update chart`.

> Updating the DevSpace Helm Chart will only add or modify files in `templates/` which are not placed into `templates/custom/`.

If you want to add custom template files in `templates/`, you should store them in `templates/custom/` to make sure that they will not be removed or replaced when running `devspace update chart`.

[Learn more about adding custom templates and manifests.](/docs/chart/customization/custom-manifests)

---
## FAQ

<details>
<summary>
### Why should I use the DevSpace Helm Chart?
</summary>
The DevSpace Helm Chart is optimized for developer productivity and provides the following benefits:
- Follows the [best practices for Helm charts](https://docs.helm.sh/chart_best_practices)
- Easy configuration for horizontal auto-scaling
- Out-of-the-box ingress connectivity via service `external`
- Simple service configuration for your deployments
- Automatic provisioning of pods as StatefulSets if volumes are attached
- Automatic provisioning of pods as Deployments if they are stateless
- Optimized for easy rollbacks if chart deployment fails
- Easy updates via `devspace update chart`
</details>
