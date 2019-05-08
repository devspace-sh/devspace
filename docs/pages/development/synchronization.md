---
title: Synchronize files
---

The code synchronization feature of DevSpace CLI allows you to use hot reloading. Especially for developers of programming languages that support hot reloading, such as nodejs, re-building and re-deploying containers woule be annoying. Therefore, DevSpace CLI uses a smart syncing mechanism that is able to sync local file changes to remote containers directly without the need of restarting the container. This greatly accelerates development, debugging and testing directly in remote containers.

## Add a path to be synchronized
You can use `devspace add sync --local=[LOCAL_PATH] --container=[CONTAINER_PATH]` to tell DevSpace CLI that `[LOCAL_PATH]` within the project on your computer and `[CONTAINER_PATH]` within your Space should be synchronized.
```bash
devspace add sync --local="./src" --container="/app"
```
The exemplary command above would configure a synchronization between the local path `./src` and the path `/app` within your container.

> Is is highly recommended to use a **relative** path within your project for the flag `--local` and an **absolute** path within your container for the `--container` flag.

Besides using the convenience command `devspace add sync`, you can also edit the configuration option in `dev.sync` within the config file `.devspace/config.yaml`. Running the exemplary command shown above would produce the following config:

```yaml
dev:
  selectors:
  - name: default
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
  sync:
  - containerPath: /app
    localSubPath: ./src
    # Use default selector defined above
    selector: default
```

The `selector` field shown above refers to the name of one of the selectors defined in `dev.selectors` and decides which container is to be selected for synchronizing files.

[Learn more about selectors.](/docs/configuration/reference#devselectors)

## Define paths to be excluded from sync
Sometimes, it is recommended to exclude certain paths from being synchronized, e.g.
- Files that change very frequently (e.g. log files)
- Files that are very large (e.g. database dumps)
- Directories containing temporary files
```yaml
dev:
  selectors:
  - name: default
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
  sync:
  - containerPath: /app
    localSubPath: ./src
    # Use default selector defined above
    selector: default
    uploadExcludePaths:
    - node_modules/
    downloadExcludePaths:
    - /app/tmp
    excludePaths:
    - Dockerfile
    - logs/
```
The above example would configure the sync, so that:
- `./src/node_modules` would not be uploaded to the container
- `/app/tmp` wiil not be downloaded from the container
- `./src/Dockerfile` and `./src/logs/` would not be synchronized at all

> Generally, the config options for excluding paths use the same syntax as `.gitignore`

## Remove sync paths
You can use the command `devspace remove sync --local=[LOCAL_PATH] --container=[CONTAINER_PATH]` to tell DevSpace CLI to remove the sync configurations where `localSubPath=[LOCAL_PATH]` and `containerPath=[CONTAINER_PATH]` from `dev.sync` in `.devspace/config.yaml`
```bash
devspace remove sync --local="./src" --container="/app"
```
This examplary command would remove the sync config created by the example command for `devspace add sync` as shown above.

## View sync status and logs
To get information about current synchronizationa activities, simply run:
```bash
devspace status sync
```
Additionally, you can ciew the sync log within `.devspace/logs/sync.log` to get more detailed information.


---
## FAQ

<details>
<summary>
### How does the sync work?
</summary>
DevSpace CLI establishes a bi-directional code synchronization between the specified local folders and the remote container folders. It automatically recognizes any changes within the specified folders during the session and will update the corresponding files locally and remotely in the background.
</details>

<details>
<summary>
### Are there any requirements for the sync to work?
</summary>
Some basic POSIX binaries have to be present in the container (which usually exist in most containers): sh, tar, cd, sleep, find, stat, mkdir, rm, cat, printf, echo, kill

Other than the binaries listed above, no server-side component for code synchronization is required, as the sync algorithm runs completely client-only within DevSpace CLI. The synchronization mechanism works with any container filesystem and no special binaries have to be installed into the containers. File watchers running within the containers like nodemon will also recognize changes made by the synchronization mechanism.
</details>

<details>
<summary>
### How does the initial sync right after `devspace dev` work?
</summary>
If synchronization is started, the sync initially compares the remote folder and the local folder and merges the contents with the following rules:
- If a file or folder exists remote, but not locally, then download file / folder
- If a file or folder exists locally, but not remote, then upload file / folder
- If a file is newer locally than remote then upload the file (The opposite case is not true, older local files are not overriden by newer remote files)
</details>


<details>
<summary>
### What is the performance impact on using the file sync?
</summary>
The sync mechanism is normally very reliable and fast. Syncing several thousand files is usually not a problem. Changes are packed together and compressed before synchronization, which improves performance especially for transferring text files. Transferring large compressed binary files is possible, however can affect performance negatively. Rename operations are currently recognized as a separate remove and create operation, which in normal workflows has at most a minor performance impact, however renaming huge folders with tens of thousands of files can impact performance negatively and should be avoided. Remote changes can sometimes have a delay of 1-2 seconds till they are downloaded, depending on how big the synchronized folder is. It should be generally avoided to sync the complete container filesystem.
</details>
