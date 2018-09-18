---
title: devspace init
---

Run `devspace init` to get your project ready to start a DevSpace.

```bash
Usage:
  devspace init [flags]

Flags:
  -h, --help                      help for init
  -l, --language string           Programming language of your project
  -o, --overwrite                 Overwrite existing chart files and Dockerfile
  -r, --reconfigure               Change existing configuration
      --templateRepoPath string   Local path for cloning chart template repository (uses temp folder
if not specified)
      --templateRepoUrl string    Git repository for chart templates (default "https://github.com/covexo/devspace-templates.git")
```

## File Structure
Running `devspace init` will create the following files for you:

```bash
YOUR_PROJECT_PATH/
|
|-- Dockerfile
|
|-- chart/
|   |-- Chart.yaml
|   |-- values.yaml
|   |-- templates/
|       |-- deployment.yaml
|       |-- service.yaml
|       |-- ingress.yaml
|
|-- .devspace/
|   |-- .gitignore
|   |-- cluster.yaml
|   |-- config.yaml
```
