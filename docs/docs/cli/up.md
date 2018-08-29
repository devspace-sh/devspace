---
title: devspace up
---

With `devspace up`, you build your image, start your DevSpace and connect to it.

```bash
Usage:
  devspace up [flags]

Flags:
  -b, --build            Build image if Dockerfile has been modified (default true)
  -h, --help             help for up
      --init-registry    Install or upgrade Docker registry (default true)
      --no-sleep         Enable no-sleep
  -o, --open string      Install/upgrade tiller (default "cmd")
      --portforwarding   Enable port forwarding (default true)
  -s, --shell string     Shell command (default: bash, fallback: sh)
      --sync             Enable code synchronization (default true)
      --tiller           Install/upgrade tiller (default true)
```

**Note**: Every time you run `devspace up`, your containers will be re-deployed. This way, you will always start with a clean state.
