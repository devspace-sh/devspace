---
title: Configure persistent volumes
---

Generally, containers are stateless. That means that whenever you write anything to the filesystem of the container, it will be removed when the container restarts.

Altough it is recommend to keep your applications as stateless as possible, sometimes you will need to store data in a directory that should "survive" a container restart, e.g. the data directory within a database container (e.g. `/var/lib/mysql` for mysql).

Persistent volumes allow you to define a virtual device which is independent of your containers an can be mounted into the containers. An analogy would be a USB stick (persistent volume) that you plug into a computer (container) which always resets the in-build hard drive on every restart.

When using the [DevSpace Helm Chart], you can edit the `chart/values.yaml` to:
1. [Define persistent volumes](#define-persistent-volumes)
2. [Mount these persistent volumes into containers](#mount-persistent-volumes)

## Define persistent volumes
You can define persistent volumes in the `volumes` section of `chart/values.yaml`.
```yaml
volumes:
- name: nginx
  size: "2Gi"
- name: mysql-data
  size: "5Gi"
```
The above example defines two volumes, one called `nginx` with size `2 Megabyte` and one called `mysql-data` with size `5 Gigabyte`.

<details>
<summary>
### View the specification for volumes
</summary>
```yaml
name: [a-z0-9-]{1,253}      # Name of the volume (used to mount the volume)
size: [number] + Gi|Mi|Ki   # Size of the volume in Gigabyte, Megabyte or Kilobyte
```
</details>

## Mount persistent volumes
You can mount persistent volumes for each `container` defined in `components[*].container[*].volumeMounts` section of `chart/values.yaml`.
```yaml
components:
- name: default
  containers:
  - image: "dscr.io/username/mysql"
    volumeMounts:
    - containerPath: /var/lib/mysql
      volume:
        name: mysql-data
        path: /mysql
        readOnly: false
```
The example above would mount the folder `/mysql` within the volume `mysql-data` into the path `/var/lib/mysql` within the first container of the component `default` and allow the container to edit the files within the mounted volume path.

<details>
<summary>
### View the specification for volume mounts
</summary>
```yaml
containerPath: [path]       # Path within the container
volume:                     # Volume to mount
  name: [volume-name]       # Name of the volume as defined in `volumes` within `chart/values.yaml`
  path: [path]              # Path within the volume
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

If you use the DevSpace Helm Chart, it will automatically deploy containers within a StatefulSet when you mount any persistent volumes. Kubernetes will not delete these persistent volumes when you delete or update the StatefulSet.
</details>

<details>
<summary>
### How can I delete everything (including persistent volumes) within a Space?
</summary>
If you want to force-delete everything (including persistent volumes) within a Space, you can run the following commands:
```bash
devspace use space [SPACE_NAME]
devspace purge
kubectl delete persistentvolumeclaims --all
```
> **Warning: The commands listed above will delete everything within your Space. All your data will be lost forever and cannot be recovered.**
</details>
