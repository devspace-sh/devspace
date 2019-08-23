---
title: Volumes
---

Generally, containers are stateless. That means that whenever you write anything to the filesystem of the container, it will be removed when the container restarts.

Altough it is recommend to keep your applications as stateless as possible, sometimes you will need to store data in a directory that should "survive" a container restart, e.g. the data directory within a database container (e.g. `/var/lib/mysql` for mysql).

Persistent volumes allow you to define a virtual device which is independent of your containers an can be mounted into the containers. An analogy would be a USB stick (persistent volume) that you plug into a computer (container) which always resets the in-build hard drive on every restart.

When you want to persist a folder within a container of one of your components, you need to:
1. [Define a volume](#define-persistent-volumes) within this component
2. [Mount this volume into the container](#mount-persistent-volumes)

> DevSpace components require you to specify separete volumes for each component, i.e. the volumes of a component can only be mounted by the containers of the same component.

## Define volumes
DevSpace components allow you to define the following types of Kubernetes volumes:
- [Persistent volumes](#define-persistent-volumes)
- [ConfigMap volumes](#define-configmap-volumes)
- [Secret volumes](#define-secret-volumes)

### Define Persistent Volumes
In the `devspace.yaml`, you can define persistent volumes in the `volumes` section of each component deployment:
```yaml
deployments:
- name: my-component
  component:
    volumes:
    - name: nginx
      size: "2Gi"
    - name: mysql-data
      size: "5Gi"
```
The above example defines two volumes, one called `nginx` with size `2 Gigabyte` and one called `mysql-data` with size `5 Gigabyte`.

<details>
<summary>
#### Show specification for Persistent Volumes
</summary>
```yaml
volumes:
- name: [a-z0-9-]{1,253}        # Name of the volume (used to mount the volume)
  size: [number] + Gi|Mi|Ki     # Size of the volume in Gigabyte, Megabyte or Kilobyte
```
</details>

### Define ConfigMap Volumes
Using DevSpace components, you can define Kubernetes ConfigMaps as volumes according to the [Kubernetes ConfigMapVolumeSource specification](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#configmapvolumesource-v1-core):
```yaml
deployments:
- name: my-component
  component:
    volumes:
    - name: nginx-config
      configMap:
        name: my-configmap
```

<details>
<summary>
#### Show specification for ConfigMap Volumes
</summary>
```yaml
volumes:
- name: [a-z0-9-]{1,253}        # Name of the volume (used to mount the volume)
  configMap:                    # Kubernetes ConfigMapVolumeSource v1
    name: [a-z0-9-]{1,253}      # Name of the ConfigMap
    ...
```
</details>

### Define Secret Volumes
Using DevSpace components, you can define Kubernetes Secrets as volumes according to the [Kubernetes SecretVolumeSource specification](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#secretvolumesource-v1-core):
```yaml
deployments:
- name: my-component
  component:
  volumes:
  - name: secret-token
    secret:
      secretName: my-secret
```

<details>
<summary>
#### Show specification for Secret Volumes
</summary>
```yaml
volumes:
- name: [a-z0-9-]{1,253}            # Name of the volume (used to mount the volume)
  secret:                           # Kubernetes SecretVolumeSource v1
    secretName: [a-z0-9-]{1,253}    # Name of the Secret
    ...
```
</details>

## Mount volumes into containers
After defining a volume for a component, you can mount it in the containers of the same component within the `volumeMounts` section:
```yaml
deployments:
- name: my-component
  component:
    containers:
    - image: "dscr.io/username/mysql"
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
The example above would create a volume called `mysql-data` for the component `my-component` and mount the folder `/mysql` within this volume into the path `/var/lib/mysql` within a container of this component. By mounting this volume to `/var/lib/mysql`, you allow the container to edit the files and folder contained within `/var/lib/mysql` and restart without losing these changes.

<details>
<summary>
#### View the specification for volume mounts
</summary>
```yaml
containerPath: [path]       # Path within the container
volume:                     # Volume to mount
  name: [volume-name]       # Name of the volume as defined in `volumes` within `chart/values.yaml`
  subPath: [path]           # Path within the volume
  readOnly: false|true      # Detault: false | set to true for read-only mounting
```
</details>


## Delete persistent volumes
By default, persisent volumes will not be deleted automatically when you remove them from the `volumes` section of `chart/values.yaml`. This ensures that persistent data is not being deleted on accident.

To remove a persistent volume, you have to remove it manually with the following commands:
```bash
devspace add space [SPACE_NAME]
kubectl delete persistentvolumeclaim [VOLUME_NAME]
```
> **Warning: Deleting persistent volumes cannot be undone. Your data will be lost forever and cannot be recovered.**

---
## FAQ

<details>
<summary>
### Can I mount one persistent volume within multiple containers?
</summary>
**Yes, but** only if the containers are either in the same component or if at most one of the containers mounts the volume with the `readOnly: false` option (e.g. one container with `readOnly: false` and 3 other containers with `readOnly: true` would work).
</details>

<details>
<summary>
### Will my persistent volumes be deleted when I re-deploy my application?
</summary>
Generally: **No.**

The [DevSpace Component Chart](/docs/deployment/components/what-are-components#devspace-component-helm-chart) used to deploy DevSpace components will automatically deploy containers as part of a StatefulSet when you mount any persistent volumes. Kubernetes will not delete these persistent volumes when you delete or update the StatefulSet.
</details>

<details>
<summary>
### How can I delete everything (including persistent volumes) within a Space?
</summary>
If you want to force-delete everything (including persistent volumes) within a Space, you can run the following commands:
```bash
devspace purge
kubectl delete persistentvolumeclaims --all
```
> **Warning: The commands listed above will delete everything within your Space. All your data will be lost forever and cannot be recovered.**
</details>
