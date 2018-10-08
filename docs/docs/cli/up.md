---
title: devspace up
---

With `devspace up`, you build your image, start your DevSpace and connect to it.  

The command will do the following:  

1. Ensure that a tiller server is available (if not it will automatically deploy one to the specified namespace)
2. Optionally it will deploy a docker registry if this was desired
3. Build the docker image if changed or forced by -b
  * Push the built image to the specified registry
5. Redeploy the chart if release was not found, image was rebuilt or -d option was specified
6. Establish port forwarding and sync
7. Execute the specified command in the container (default: open a terminal)

```bash
Usage:
  devspace up [flags]

Flags:
  -b, --build              Force image build
  -c, --container string   Container name where to open the shell
  -d, --deploy             Force chart deployment
  -h, --help               help for up
      --init-registries    Initialize registries (and install internal one) (default true)
      --no-sleep           Enable no-sleep (Override the containers.default.command and containers.default.args values with empty strings)
      --portforwarding     Enable port forwarding (default true)
      --sync               Enable code synchronization (default true)
      --tiller             Install/upgrade tiller (default true)
      --verbose-sync       When enabled the sync will log every file change 

Examples:
devspace up         # Start the devspace
devspace up bash    # Execute bash command after deploying
```

**Note**: Every time you run `devspace up`, your containers will be re-deployed. This way, you will always start with a clean state.
