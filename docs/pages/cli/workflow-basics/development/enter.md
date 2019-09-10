---
title: Start terminal sessions
---

Entering a container can help you debug issues within the container.
```bash
devspace enter
```

## Open terminals for different containers
If you want to open a terminal for a container other than your default container, you can specify flags like `-l / --label-selector` or `--selector`. Alternatively, you can use the `-p / --pick` flag to get a list of available containers.
```bash
devspace enter -p
```
Running this command will give you a list of containers that you can open a terminal for.

## Run a command after opening the terminal
If you provide arguments to `devspace enter`, DevSpace will execute the arguments string as a command inside the container instead of opening a terminal.
```bash
devspace enter [command]
```
Running this command will give you a list of containers that you can open a terminal for.

Example: `devspace enter echo "Hello World!"` would provide a similar output like this one:
```bash
$ devspace enter echo "Hello World"
[info]   Loaded config from devspace.yaml     
[info]   Opening shell to pod:container devspace-app-b6b4548ff-hbchq:container-0
Hello World!
```
