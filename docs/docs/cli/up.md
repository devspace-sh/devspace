---
title: devspace up
---

With `devspace up`, you build your image, start your DevSpace and connect to it. 

```bash
Usage:
  devspace up [flags]

Flags:
  -b, --build              Build image if Dockerfile has been modified (default true)
  -c, --container string   Container name where to open the shell
  -d, --deploy             Deploy chart
  -h, --help               help for up
      --init-registries    Initialize registries (and install internal one) (default true)
      --no-sleep           Enable no-sleep 
      --portforwarding     Enable port forwarding (default true)
      --sync               Enable code synchronization (default true)
      --tiller             Install/upgrade tiller (default true)
      --verbose-sync       When enabled the sync will log every file change 

Examples:
devspace up         # Start the devspace
devspace up bash    # Execute bash command 
```

**Note**: Every time you run `devspace up`, your containers will be re-deployed. This way, you will always start with a clean state.
