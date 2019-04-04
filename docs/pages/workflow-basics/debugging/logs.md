---
title: View logs
---

To view the logs of a container, run this command:
```bash
devspace logs
```
By default, this will show the last 200 log lines of the first container within your `default` sector. 

## Show logs of different containers
If you want to access the logs of a container other than your default container, you can specify flags like `-l / --label-selector` or `--selector`. Alternatively, you can use the `-p / --pick` flag to get a list of available containers.
```bash
devspace logs -p
```

## Stream logs in real-time
To stream the logs of a container in real-time, use the `-f / --follow` flag for `devspace logs`.
```bash
devspace logs -f
```
