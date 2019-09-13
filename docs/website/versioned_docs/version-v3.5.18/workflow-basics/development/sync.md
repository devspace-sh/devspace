---
title: Synchronize files on-demand
id: version-v3.5.18-sync
original_id: sync
---

DevSpace CLI is able to synchronize files from the local computer into any running pod on-demand. This can be useful for debugging to change container files on the fly without constantly running `kubectl cp` after every file change.  

Run the following command to start syncing files between the local computer
```bash
devspace sync
```

The command starts synchronizing files from the current directory with files in any chosen containers working directory. 
`devspace sync` does the following during synchronization:
1. Inject a small helper binary via `kubectl cp` into the target container
2. Download all files that are not found locally
3. Upload all files that are not found remotely
4. Watches locally and remotely for changes and uploads or downloads them

There are many command parameters how you can modify the behaviour of `devspace sync`, e.g. excluding files or changing the remote or local path:
```html
Usage:
  devspace sync [flags]

Flags:
  -c, --container string        Container name within pod where to execute command
      --container-path string   Container path to use (Default is working directory)
  -e, --exclude strings         Exclude directory from sync
  -h, --help                    help for sync
  -l, --label-selector string   Comma separated key=value selector list (e.g. release=test)
      --local-path string       Local path to use (Default is current directory (default ".")
  -n, --namespace string        Namespace where to select pods
  -p, --pick                    Select a pod 
      --pod string              Pod to open a shell to
  -s, --selector string         Selector name (in config) to select pod/container for terminal
      --verbose                 Shows every file that is synced
```

You can also tell DevSpace CLI to start automatically synchronizing files on `devspace dev`. Take a look at [synchronizing files](/docs/development/synchronization) for more information.
