---
title: Configuring File Synchronization
sidebar_label: File Sync
---

The code synchronization feature of DevSpace CLI allows you to use hot reloading during development. Especially for developers of programming languages that support hot reloading, such as nodejs, re-building and re-deploying containers is very annoying and time consuming. Therefore, DevSpace CLI uses a smart syncing mechanism that is able to sync local file changes to remote containers directly without the need of rebuilding the container. This greatly accelerates development, debugging and testing directly in remote containers.

## Synchronizing files with DevSpace

DevSpace CLI provides a convenient command `devspace sync`, which starts synchronizing files from the current directory with files in any chosen containers working directory. 
`devspace sync` does the following during synchronization:
1. Inject a small helper binary via `kubectl cp` into the target container
2. Download all files that are not found locally
3. Upload all files that are not found remotely
4. Watches locally and remotely for changes and uploads or downloads them

There are many command parameters how you can modify the behaviour of `devspace sync`, e.g. excluding files or changing the remote or local path. If you want to start synchronization automatically during `devspace dev` you can add a sync configuration in your `devspace.yaml` like this:

```yaml
dev:
  sync:
  - localSubPath: ./src
    # Select pods by following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
    # Start syncing to the containers current working directory (You can also use absolute paths)
    containerPath: .
```

This tells DevSpace to automtically start synchronzing files as soon as you run `devspace dev`. The `labelSelector` option tells DevSpace which pods to select for synchronization.

## Define paths to be excluded from sync
Sometimes, it is recommended to exclude certain paths from being synchronized, e.g.
- Files that change very frequently (e.g. log files)
- Files that are very large (e.g. database dumps)
- Directories containing temporary files

```yaml
dev:
  sync:
  - containerPath: .
    localSubPath: ./src
    # This tells devspace to select pods that have the following labels
    labelSelector:
      app.kubernetes.io/component: default
      app.kubernetes.io/name: devspace-app
    # Only download changes to these paths, but do not upload any changes (.gitignore syntax)
    uploadExcludePaths:
    - node_modules/
    # Only upload changes to these paths, but do not download any changes (.gitignore syntax)
    downloadExcludePaths:
    - /app/tmp
    # Ignore these paths completely during synchronization (.gitignore syntax)
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
You can use the command `devspace remove sync --local=[LOCAL_PATH] --container=[CONTAINER_PATH]` to tell DevSpace CLI to remove the sync configurations where `localSubPath=[LOCAL_PATH]` and `containerPath=[CONTAINER_PATH]` from `dev.sync` in `devspace.yaml`
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
DevSpace CLI establishes a bi-directional code synchronization between the specified local folders and the remote container folders. It automatically recognizes any changes within the specified folders during the session and will update the corresponding files locally and remotely in the background. It uses a small helper binary that is injected into the target container to accomplish this.
</details>

<details>
<summary>
### Are there any requirements for the sync to work?
</summary>
The `tar` command has to be present in the container otherwise `kubectl cp` does not work and the helper binary cannot be injected into the container.  

Other than that, no server-side component or special container privileges for code synchronization are required, as the sync algorithm runs completely client-only within DevSpace CLI. The synchronization mechanism works with any container filesystem and no special binaries have to be installed into the containers. File watchers running within the containers like nodemon will also recognize changes made by the synchronization mechanism.
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
The sync mechanism is normally very reliable and fast. Syncing several thousand files is usually not a problem. Changes are packed together and compressed during synchronization, which improves performance especially for transferring text files. Transferring large compressed binary files is possible, however can affect performance negatively. Remote changes can sometimes have a delay of 1-2 seconds till they are downloaded, depending on how big the synchronized folder is. It should be generally avoided to sync the complete container filesystem.
</details>
