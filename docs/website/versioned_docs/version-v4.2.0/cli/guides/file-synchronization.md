---
title: How To Synchronize Files From/To Containers
sidebar_label: Sync Files From/To Containers
id: version-v4.2.0-file-synchronization
original_id: file-synchronization
---

There are two types of file synchronzation processes:
- on-demand file sync using `devspace dev` and
- file synchronization during development mode using `devspace dev`. 

## Start On-Demand File Sync
To establish an on-demand file synchronization between your local computer and the containers running inside Kubernetes, use the following command:
```bash
devspace sync # optional flags: --local-path=./ --container-path=/some/path --no-watch
```
[View the specification of the `devspace sync` command.](../../cli/commands/devspace_sync)

[Learn more about the `devspace sync` command.](https://devspace.cloud/blog/2019/10/18/release-devspace-v4.1.0-kubectl-cp-file-synchronization)

## Configure File Sync
If you want to start file synchronization every time you run `devspace dev`, you can configure it within `devspace.yaml`.

[Learn more about configuring file synchronization using `devspace.yaml`.](../../cli/development/configuration/file-synchronization)
