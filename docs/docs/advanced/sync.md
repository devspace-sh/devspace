---
title: Source Code Synchronization
---

# Code Synchronization
The DevSpace CLI synchronization mechanism is written from scratch and made for kubernetes development with hot reloading. While current other kubernetes development solutions (like draft, skaffold and telepresence) help the developer with the problem how to test and deploy cloud-native applications in kubernetes clusters, actual iterative development, testing and hot reloading of cloud-native applications within kubernetes is still an annoying problem. Especially for developers of programming languages that support hot reload development environments, such as nodejs, this is a hassle. The DevSpace CLI has a syncing mechanism that is able to sync local file changes to remote containers directly without the need of restarting the container, a deployment pipeline or a docker build. This greatly accelerates development, debugging and testing directly in remote containers.

Our main requirements to the sync mechanism were:
- Easy to integrate, no cluster dependencies, client-only implementation
- Should work in every kubernetes cluster
- Should work with most containers, without installing additional binaries or changing the Dockerfile
- Should work with every container filesystem (i.e. ephemeral storage and mounted volumes)
- File watchers and hot reload tools like nodemon should recognize sync changes like on local filesystem
- Fast and reliable

 If synchronization is configured (check with `devspace list sync`), the DevSpace CLI will establish a bi-directional code synchronization between the specified local folders and the remote container folders. It automatically recognizes any changes within the specified folders during the session and will update the corresponding files locally and remotely in the background. You can check the latest sync activity by running the command `devspace status sync` or take a look at the `sync.log` in `.devspace/logs`.

## Sync Requirements
No server-side component for code synchronization is required, the sync is client-only. The synchronization mechanism works with any container filesystem and no special binaries have to be installed into the containers. File watchers running within the containers like nodemon will also recognize changes made by the synchronization mechanism.

Some basic POSIX binaries have to be present in the container (which usually exist in most containers): `sh, tar, cd, sleep, find, stat, mkdir, rm, cat, printf, echo, kill`

## Excluding Files and Folders
You are able to fully or partly exclude certain files and folders from synchronization. Take a look at [.devspace/config.yaml](/docs/configuration/config.yaml.html) for more information where to specify the ignore rules in the configuration. The exclude path syntax is the [.gitignore](https://git-scm.com/docs/gitignore) syntax. 

There are 3 different options for each sync path to exclude files and folders:
1. excludePaths: Matched paths are completely ignored during synchronization
2. downloadExcludePaths: Local changes are uploaded, but remote changes are not downloaded 
3. uploadExcludePaths: Local changes are not uploaded, but remote changes are downloaded

## Initial Sync
If synchronization is started, the sync initially compares the remote folder and the local folder and merges the contents with the following rules:
- If a file or folder exists remote, but not locally, then download file / folder
- If a file or folder exists locally, but not remote, then upload file / folder
- If a file is newer locally than remote then upload the file (The opposite case is not true, older local files are not overriden by newer remote files)

## Performance Notes
The sync mechanism is normally very reliable and fast. Syncing several thousand files is usually not a problem. Changes are packed together and compressed before synchronization, which improves performance especially for transferring text files. Transferring large compressed binary files is possible, however can affect performance negatively. Rename operations are currently recognized as a separate remove and create operation, which in normal workflows has at most a minor performance impact, however renaming huge folders with tens of thousands of files can impact performance negatively and should be avoided. Remote changes can sometimes have a delay of 1-2 seconds till they are downloaded, depending on how big the synchronized folder is. It should be generally avoided to sync the complete container filesystem.
