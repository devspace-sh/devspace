---
title: Configure Component Deployments
sidebar_label: Components
---

To deploy components, you need to configure them within the `deployments` section of the `devspace.yaml`.
```yaml
deployments:
- name: frontend
  helm:
    componentChart: true
    values:
      containers:
      - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
      service:
        ports:
        - port: 3000
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/appbackend
      - image: mysql
        volumeMounts:
        - containerPath: /var/lib/mysql
          volume:
            name: mysql-data
            subPath: /mysql
            readOnly: false
      volumes:
      - name: mysql-data
        size: "5Gi"
```

[What are components?](../../../../cli/deployment/components/what-are-components)


## Containers &amp; Pods

### `deployments[*].component.containers`
See [Containers](../../../../cli/deployment/components/configuration/containers) for details.


### `deployments[*].component.initContainers`
The `initContainers` section allows the exact same configuration options as the `containers` section. See [Containers](../../../../cli/deployment/components/configuration/containers) for details.


### `deployments[*].component.labels`
The `labels` option expects a map with Kubernetes labels. 

By default, the component chart sets a couple of labels following the best practices described in the Kubernetes documentation:
- `app.kubernetes.io/name: devspace-app`
- `app.kubernetes.io/component: [DEPLOYMENT_NAME]`
- `app.kubernetes.io/managed-by: Helm`

> You can specify additional labels using the `labels` option but the default / best practice labels will still be set for the component.

All additional labels will be added to the pods of this component as well as to your Deployments or StatefulSets used to create the pods.

#### Default Value For `labels`
```yaml
labels: []
```

#### Example: Additional Labels
```yaml
deployments:
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/appbackend
      labels:
        label1: label-value-1
        label1: label-value-2
```


### `deployments[*].component.annotations`
The `annotations` option expects a map with Kubernetes annotations. 

By default, the component chart sets a couple of annotations following the best practices described in the Kubernetes documentation:
- `helm.sh/chart: component-chart-vX.Y.Z`

> You can specify additional annotations using the `annotations` option but the default / best practice annotations will still be set for the component.

All additional annotations will be added to the pods of this component as well as to your Deployments or StatefulSets used to create the pods.

#### Default Value For `annotations`
```yaml
annotations: []
```

#### Example: Additional Annotations
```yaml
deployments:
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/appbackend
      annotations:
        annotation1: annotation-value-1
        annotation1: annotation-value-2
```


## Volumes &amp; Persistent Storage
### `deployments[*].component.volumes`
See [Volumes](../../../../cli/deployment/components/configuration/volumes) for details.


## Service &amp; In-Cluster Networking
### `deployments[*].component.service`
See [Service](../../../../cli/deployment/components/configuration/service) for details.

### `deployments[*].component.serviceName`
The `serviceName` option expects a string that will be used as a name for the headless service if the component will be deployed as a StatefulSet instead of a Deployment. This happens automatically when one of the containers mounts a persistent volume.

#### Default Value For `serviceName`
```yaml
serviceName: "[COMPONENT_NAME]-headless"
```

#### Example: Custom Name for Headless Service
```yaml
deployments:
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: mysql
        volumeMounts:
        - containerPath: /var/lib/mysql
          volume:
            name: mysql-data
            subPath: /mysql
            readOnly: false
      volumes:
      - name: mysql-data
        size: "5Gi"
      serviceName: "custom-name-for-headless-service"
```
**Explanation:**  
Instead of the default name `backend-headless`, the headless service for the ReplicaSet created by this component would be `custom-name-for-headless-service`.


## Ingress &amp; Domain
### `deployments[*].component.ingress`
See [Ingress (Domain)](../../../../cli/deployment/components/configuration/ingress) for details.


## Scaling
### `deployments[*].component.replicas`
See [Scaling](../../../../cli/deployment/components/configuration/containers) for details.

### `deployments[*].component.autoScaling`
See [Scaling](../../../../cli/deployment/components/configuration/containers) for details.


## Advanced

### `deployments[*].component.rollingUpdate`
The `rollingUpdate` option the Kubernetes configuration for rolling updates. 

The following fields are supported for **Deployments** (none of the containers mounts any persistent volumes):
- `enabled`
- `maxSurge`
- `maxUnavailable`

The following fields are supported for **StatefulSets** (at least one of the containers mounts a persistent volume):
- `partition`

> If `enabled = true`, the Kubernetes rolling update type will be set to `RollingUpdate`. By default, this value is `Recreate`.

#### Default Value For `rollingUpdate`
```yaml
rollingUpdate:
  enabled: false
  maxSurge: "25%"
  maxUnavailable: "0%"
  partition: 0
```

#### Example: Enabling Rolling Updates
```yaml
deployments:
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/appbackend
      rollingUpdate:
        enabled: true
        maxUnavailable: "50%"
```

### `deployments[*].component.pullSecrets`
The `pullSecrets` option expects an array of strings with names of Kubernetes secrets containing image registry authentification data.

Adding a secret name to `pullSecrets` will tell DevSpace to add the secret name as pullSecret to the Deployment or StatefulSet that will be created for this component.

> DevSpace is also able to [create pull secrets for registries automatically](../../../../cli/image-building/workflow-basics#8-create-image-pull-secret). These pull secrets do **not** need to be added to `pullSecrets` as they will be added to the service account instead which makes them available to Kubernetes without adding them to each Deployment or StatefulSet.

#### Default Value For `pullSecrets`
```yaml
pullSecrets: []
```

#### Example: Custom Name for Headless Service
```yaml
deployments:
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/appbackend
      pullSecrets:
      - pull-secret-1
      - pull-secret-2
```


### `deployments[*].component.podManagementPolicy`
The `podManagementPolicy` option expects a string which sets the [`podManagementPolicy` Kubernetes attribute for a StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#pod-management-policies).

#### Default Value For `podManagementPolicy`
```yaml
podManagementPolicy: OrderedReady
```

#### Example: Custom Name for Headless Service
```yaml
deployments:
- name: backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/appbackend
      podManagementPolicy: Parallel
```


<br>

---
## Useful Commands

### `devspace list available-components`
Run the following command to get a list of predefined components:
```bash
devspace list available-components
```
Use [`devspace add deployment [NAME] --component=[COMPONENT_NAME]`](#devspace-add-deployment-name-component-mysql-redis) to add one of these components to the `deployments` section in `devspace.yaml`.


### `devspace add deployment [NAME] --component=mysql|redis|...`
Run the following command to add a predefined component to the `deployments` section in `devspace.yaml`:
```bash
devspace add deployment [deployment-name] --component=[component-name]
```
Example: `devspace add deployment database --component=mysql`

> Running `devspace add deployment` only adds a deployment to `devspace.yaml` but does not actually deploy anything. To deploy the newly added deployment, run `devspace deploy` or `devspace dev`.

#### List of predefined components
DevSpace provides the following predefined components:
- mariadb
- mongodb
- mysql
- postgres
- redis


### `devspace add deployment [NAME] --dockerfile=[PATH]`
Run one of the following commands to add a component deployment to the `deployments` section in `devspace.yaml` based on an existing Dockerfile:
```bash
devspace add deployment [deployment-name] --dockerfile="./path/to/Dockerfile"
devspace add deployment [deployment-name] --dockerfile="./path/to/Dockerfile" --image="my-registry.tld/[username]/[image]"
```
Both commands would add a component deployment to the `deployments` and a image to the `images` section defining how the image would be built, tagged and pushed using the Dockerfile you provided.

The difference between the first command and the second one is that the second one specifically defines where the Docker image should be pushed to after building the Dockerfile. Using the first command, DevSpace would assume that you want to use a private repository at [dscr.io](../../../../cloud/images/dscr-io), the free image registry sponsored by DevSpace Cloud.

> If you are using a private Docker registry, make sure to [login to this registry](../../../../cli/image-building/workflow-basics#registry-authentication).

> Running `devspace add deployment` only adds a deployment to `devspace.yaml` but does not actually deploy anything. To deploy the newly added deployment, run `devspace deploy` or `devspace dev`.


### `devspace add deployment [NAME] --image=[IMAGE]`
Run the following command to add a component deployment to the `deployments` section in `devspace.yaml` based on an existing Docker image:
```bash
devspace add deployment [deployment-name] --image="my-registry.tld/[username]/[image]"
```
This command would add a component deployment to the `deployments` section in `devspace.yaml`.

> If you are using a private Docker registry, make sure to [login to this registry](../../../../cli/image-building/workflow-basics#registry-authentication).

> Running `devspace add deployment` only adds a deployment to `devspace.yaml` but does not actually deploy anything. To deploy the newly added deployment, run `devspace deploy` or `devspace dev`.

#### Example using Docker Hub
```bash
devspace add deployment database --image="mysql"
```


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
