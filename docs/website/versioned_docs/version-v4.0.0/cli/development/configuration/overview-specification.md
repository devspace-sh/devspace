---
title: Configuring Development Mode
sidebar_label: Configuration
id: version-v4.0.0-overview-specification
original_id: overview-specification
---

You can start the development mode for your project using `devspace dev`.

The configuration options for the development mode are located in the `dev` section of the `devspace.yaml`.
```yaml
dev:
  ports: ...        # Port-Forwarding to make your application accessible on localhost
  open: ...         # Auto-Open to open URLs after deploying a project in development mode
  sync: ...         # File Synchronzation to sync files between local workspace and remote containers
  logs: ...         # Log Streaming to configure which container logs should be streamed
  autoReload: ...   # Auto-Reload to automatically redeploy your project when major changes occur to specific files
  interactive: ...  # Interactive Mode for starting containers in sleep mode and opening an interactive terminal session.
```

Take a look at the following pages for details on how to configure each section of the `dev` config:
- **[`ports` Port-Forwarding](/docs/cli/development/configuration/port-forwarding)**
- **[`open` Auto-Open](/docs/cli/development/configuration/auto-open)**
- **[`sync` File Synchronization](/docs/cli/development/configuration/file-synchronization)**
- **[`logs` Multi-Container Log Streaming](/docs/cli/development/configuration/logs-streaming)**
- **[`autoReload` Auto-Reload / Redeploy](/docs/cli/development/configuration/auto-reloading)**
- **[`interactive` Interactive Mode](/docs/cli/development/configuration/interactive-mode)**
