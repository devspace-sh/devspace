---
title: Configuring File Synchronization
sidebar_label: File Sync
id: version-v4.0.2-file-synchronization
original_id: file-synchronization
---

The code synchronization feature of DevSpace allows you to use hot reloading during development. Especially when using programming languages and frameworks that support hot reloading with tools like nodemon, re-building and re-deploying containers is very annoying and time consuming. Therefore, DevSpace uses a smart syncing mechanism that is able to sync local file changes to remote containers directly without the need of rebuilding or restarting the container.

When starting the development mode, DevSpace starts the file sync as configured in the `dev.sync` section of the `devspace.yaml`.
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - image: john/devbackend
    - image: john/debugger
dev:
  sync:
  - imageName: backend
    localSubPath: ./
    containerPath: /app
    excludePaths:
    - node_modules/
    - logs/
```

Every sync configuration consists of two essential parts:
- [Container Selection via `imageName` or `labelSelector`](#container-selection)
- [Sync Path Mapping via `localSubPath` and `containerPath`](#sync-path-mapping)

Additionally, there are several advanced options:
- [Configuring Exclude Paths via `excludePaths`, `downloadExcludePaths` and `uploadExcludePaths`](#exclude-paths)
- [Configuring Initial Sync via `waitInitialSync`](#initial-sync)
- [Configuring Network Bandwith Limits via `bandwidthLimits`](#network-bandwidth-limits)

## Container Selection
The following config options are needed to determine the container which the file synchronization should be established.

> You can set **either** `labelSelector` (optionally in combination with `containerName`) **or** `imageName`. Both options can be combined with the optional `namespace` option if needed.


### `dev.sync[*].imageName`
The `imageName` option expects a string with the name of an image from the `images` section of the `devspace.yaml`. Using `imageName` tells DevSpace to select the container based on the referenced image that was last built using DevSpace.

> Using `imageName` is not possible if multiple deployments use the same image that belongs to this `imageName` referencing the `images` section of the `devspace.yaml`.

> You cannot use the `imageName` option in combination with `labelSelector`.

#### Example: Select Container by Image
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - name: container-0
      image: john/devbackend
    - name: container-1
      image: john/debugger
dev:
  sync:
  - imageName: backend
    excludePaths:
    - node_modules/
    - logs/
  - imageName: backend-debugger
    localSubPath: ./debug-logs
    containerPath: /logs
```
**Explanation:**  
- The above example defines two images that can be used as `imageName`: `backend` and `backend-debugger`
- The deployment starts two containers and each of them uses an image from the `images` section.
- The `imageName` option of the first sync configuration in the `dev.sync` section references `backend`. That means DevSpace would select the first container for file synchronzation, as this container uses the `image: john/devbackend` which belongs to the `backend` image as defined in the `images` section.
- The first sync configuration does not define `localSubPath`, so it defaults to the project's root directory (location of `devspace.yaml`).
- The first sync configuration does not define `containerPath`, so it defaults to the container's working directory (i.e. `WORKDIR`).
- The `imageName` option of the second sync configuration in the `dev.sync` section references `backend-debugger`. That means DevSpace would select the second container for file synchronization, as this container uses the `image: john/debugger` which belongs to the `backend-debugger` image as defined in the `images` section.

In consequence, the following sync processes would be started when using the above config example assuming the local project root directoy `/my/project/`:
1. `localhost:/my/project/` forwards to `container-0:$WORKDIR` **\***
2. `localhost:/my/project/debug-logs/` forwards to `container-1:/logs`

**\* Changes on either side (local and container filesystem) that occur within the sub-folders `node_modules/` and `logs/` would be ingored.**

### `dev.sync[*].labelSelector`
The `labelSelector` option expects a key-value map of strings with Kubernetes labels.

> You cannot use the `labelSelector` option in combination with `imageName`.

#### Example: Select Container by Label
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - name: container-0
      image: john/devbackend
    - name: container-1
      image: john/debugger
dev:
  sync:
  - labelSelector:
      app.kubernetes.io/name: devspace-app
      app.kubernetes.io/component: app-backend
      custom-label: custom-label-value
    containerName: container-0
    localSubPath: ./src
    containerPath: /app/src
```
**Explanation:**  
- The `labelSelector` would select the pod created for the component deployment `app-backend`.
- Because the selected pod has two containers, we also need to specify the `containerName` option which defines the container that should be used for the file synchronization.

### `dev.sync[*].containerName`
The `containerName` option expects a string with a container name. This option is used to decide which container should be selected when using `labelSelector` option because `labelSelector` selects a pod and a pod can have multiple containers.

> The `containerName` option is not required when the pod you are selecting using `labelSelector` has only one container.

#### Example
**See "[Example: Select Container by Label](#example-select-container-by-label)"**


### `dev.sync[*].namespace`
The `namespace` option expects a string with a Kubernetes namespace used to select the container from.

> It is generally not needed to specify the `namespace` option because by default, DevSpace uses the default namespace of your current kube-context which is usually the one that has been used to deploy your containers to.


## Sync Path Mapping

### `dev.sync[*].localSubPath`
The `localSubPath` option expects a string with a path that is relative to the location of `devspace.yaml`.

#### Default Value For `localSubPath`
```yaml
localSubPath: ./ # Project root directory (folder containing devspace.yaml)
```

#### Example
**See "[Example: Select Container by Image](#example-select-container-by-image)"**


### `dev.sync[*].containerPath`
The `containerPath` option expects a string with an absolute path on the container filesystem.

#### Default Value For `containerPath`
```yaml
containerPath: $WORKDIR # working directory, set as WORKDIR in the Dockerfile
```

#### Example
**See "[Example: Select Container by Image](#example-select-container-by-image)"**



## Exclude Paths

> The config options for excluding paths use the same syntax as `.gitignore`.

> An exclude path that matches a folder recursively excludes all files and sub-folders within this folder.

### `dev.sync[*].excludePaths`
The `excludePaths` option expects an array of strings with paths that should not be synchronized between the local filesystem and the remote container filesystem.

#### Default Value For `excludePaths`
```yaml
excludePaths: [] # Do not exclude anything from file synchronization
```

#### Example: Exclude Paths from Synchronization
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - image: john/devbackend
    - image: john/debugger
dev:
  sync:
  - imageName: backend
    excludePaths:
    - logs/
    - more/logs/
    uploadExcludePaths:
    - node_modules/
    downloadExcludePaths:
    - tmp/
```
**Explanation:**  
- Files in `logs/` and in `mode/logs/` would not be synchronized at all.
- Files in `node_modules/` would only be synchroniyed from the container to the local filesystem but not the other way around.
- Files in `tmp/` would only be synchroniyed from the local to the container filesystem but not the other way around.


### `dev.sync[*].downloadExcludePaths`
The `downloadExcludePaths` option expects an array of strings with paths that should not be synchronized from the remote container filesystem to the local filesystem.

#### Default Value For `downloadExcludePaths`
```yaml
downloadExcludePaths: [] # Do not exclude anything from file synchronization
```

#### Example
**See "[Example: Exclude Paths from Synchronization](#example-exclude-paths-from-synchronization)"**

### `dev.sync[*].uploadExcludePaths`
The `uploadExcludePaths` option expects an array of strings with paths that should not be synchronized from the local filesystem to the remote container filesystem.

> This option is often useful if you want to download a dependency folder (e.g. `node_modules/`) for code completion but you never want to upload anything from there because of compiled binaries that are not portable between local filesystem and container filesystem (e.g. when your local system is Windows but your containers run Linux).

#### Default Value For `uploadExcludePaths`
```yaml
uploadExcludePaths: [] # Do not exclude anything from file synchronization
```

#### Example
**See "[Example: Exclude Paths from Synchronization](#example-exclude-paths-from-synchronization)"**


## Initial Sync

### `dev.sync[*].downloadOnInitialSync`
The `downloadOnInitialSync` option expects a boolean which defines if DevSpace should (during the initial sync procedure) download files which only exist in the container filesystem but do **not** exist on the local filesystem.

> Files listed under `excludePaths` or `downloadExcludePaths` will not be synchronized.

> By default, DevSpace removes files which do not exist on the local filesystem but are present within the container. This does not apply to files listed under `excludePaths` or `uploadExcludePaths`.

#### Default Value For `downloadOnInitialSync`
```yaml
downloadOnInitialSync: false # Do not download any files during initial sync
```

#### Example: Download Files During Initial Sync
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - image: john/devbackend
    - image: john/debugger
dev:
  sync:
  - imageName: backend
    excludePaths:
    - node_modules/*
  - imageName: backend
    localSubPath: ./node_modules/
    containerPath: /app/node_modules/
    downloadOnInitialSync: true
```
**Explanation:**  
With the configuration `devspace dev` would do the following:
- DevSpace would start port-forwarding and file synchronzation.
- Initial sync would be started automatically.
- The first sync config section would synchronize all files except files within `node_modules/`. This means that during initial sync, all remote files that are not existing locally would be deleted and other files would be updated to the most recent version.
- The second sync config section would only synchronize files within `node_modules/` and with defining `downloadOnInitialSync: true`, DevSpace would also download all remote files which are not present on the local filesystem rather than removing them.


### `dev.sync[*].waitInitialSync`
The `waitInitialSync` option expects a boolean which defines if DevSpace should wait until the initial sync process has terminated before opening the container terminal or the multi-container log streaming.

#### Default Value For `waitInitialSync`
```yaml
waitInitialSync: false # Start container terminal or log streaming before initil sync is completed
```

#### Example: Wait For Initial Sync To Complete
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - image: john/devbackend
    - image: john/debugger
dev:
  sync:
  - imageName: backend
    waitInitialSync: true
```
**Explanation:**  
With the configuration `devspace dev` would do the following:
- DevSpace would start port-forwarding and file synchronzation.
- Initial sync would be started automatically.
- After the initial sync process is finished, DevSpace starts the multi-container log streaming.


## Network Bandwidth Limits
Sometimes it is useful to throttle the file synchronization, especially when large files or a large number of files are expected to change during development. The following config options provide these capabilities:

### `dev.sync[*].bandwidthLimits.download`
The `bandwidthLimits.download` option expects an integer representing the max file download speed in KB/s, e.g. `download: 100` would limit the file sync to a download speed of `100 KB/s`.

> By default, the file synchronization algorithm uses the maximum bandwidth possible to make the sync process as fast as possible.

#### Example: Limiting Network Bandwidth
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  component:
    containers:
    - image: john/devbackend
    - image: john/debugger
dev:
  sync:
  - imageName: backend
    bandwidthLimits:
      download: 200
      upload: 100
```
**Explanation:**  
- Downloading files from the container to the local filesystem would be limited to a transfer speed of `200 KB/s`.
- Upload files from the local filesystem to the container would be limited to a transfer speed of `100 KB/s`.

### `dev.sync[*].bandwidthLimits.upload`
The `bandwidthLimits.upload` option expects an integer representing the max file upload speed in KB/s, e.g. `upload: 100` would limit the file sync to a upload speed of `100 KB/s`.

> By default, the file synchronization algorithm uses the maximum bandwidth possible to make the sync process as fast as possible.

#### Example
**See "[Example: Limiting Network Bandwidth](#example-limiting-network-bandwidth)"**


<br>

---
## Useful Commands

### `devspace add sync --local=[PATH] --container=[PATH]`
Use the convenience command `devspace add sync --local=[PATH] --container=[PATH]` to tell DevSpace to add another sync path mapping to the `dev.sync` section of the `devspace.yaml`.
```bash
devspace add sync --local=src/ --container=/app/src
```
The example above would tell DevSpace to add a sync configuration for synchronizing the local path `./src/` with the container path `/app/src`.


### `devspace status sync`
To get information about current synchronization activities, simply run:
```bash
devspace status sync
```
Additionally, you can ciew the sync log within `.devspace/logs/sync.log` to get more detailed information.


### `devspace sync`
If you want to start file synchronzation on-demand without having to configure it in `devspace.yaml` and without starting port-forwarding or log streaming etc, you can use the `devspace sync` command as shown in the examples below:
```bash
# Select pod with a picker
devspace sync --local-path=subfolder --container-path=/app

# Select pod and container by name and use current working directory as local-path
devspace sync --pod=my-pod --container=my-container --container-path=/app
```


---
## FAQ

<details>
<summary>

### How does the sync work?

</summary>

DevSpace establishes a bi-directional code synchronization between the specified local folders and the remote container folders. It automatically recognizes any changes within the specified folders during the session and will update the corresponding files locally and remotely in the background. It uses a small helper binary that is injected into the target container to accomplish this.

The algorithm roughly works like this:
1. Inject a small helper binary via `kubectl cp` into the target container
2. Download all files that are not found locally
3. Upload all files that are not found remotely
4. Watches locally and remotely for changes and uploads or downloads them

</details>

<details>
<summary>

### Are there any requirements for the sync to work?

</summary>
The `tar` command has to be present in the container otherwise `kubectl cp` does not work and the helper binary cannot be injected into the container.  

Other than that, no server-side component or special container privileges for code synchronization are required, as the sync algorithm runs completely client-only within DevSpace. The synchronization mechanism works with any container filesystem and no special binaries have to be installed into the containers. File watchers running within the containers like nodemon will also recognize changes made by the synchronization mechanism.
</details>

<details>
<summary>

### How does the initial sync right after `devspace dev` work?

</summary>
If synchronization is started, the sync initially compares the remote folder and the local folder and merges the contents with the following rules:
- If a file or folder exists locally, but not remote, then upload file / folder
- If a file is newer locally than remote then upload the file (The opposite case is not true, older local files are not overriden by newer remote files)
- If a file or folder exists on the remote filesystem, but not locally, then remove the remote file / folder (if `downloadOnInitialSync: false` which is the default configuration)
- If a file or folder exists on the remote filesystem, but not locally, then download the remote file / folder (if `downloadOnInitialSync: true`)
</details>

<details>
<summary>

### What is the performance impact on using the file sync?

</summary>
The sync mechanism is normally very reliable and fast. Syncing several thousand files is usually not a problem. Changes are packed together and compressed during synchronization, which improves performance especially for transferring text files. Transferring large compressed binary files is possible, however can affect performance negatively. Remote changes can sometimes have a delay of 1-2 seconds till they are downloaded, depending on how big the synchronized folder is. It should be generally avoided to sync the complete container filesystem.
</details>
