---
title: devspace init
---

Run `devspace init` to get your project ready to start a DevSpace.

```
Usage:
  devspace init [flags]

Flags:
  -h, --help                      help for init
  -l, --language string           Programming language of your project
  -o, --overwrite                 Overwrite existing chart files and Dockerfile
  -r, --reconfigure               Change existing configuration
      --templateRepoPath string   Local path for cloning chart template repository (uses temp folder if not specified)
      --templateRepoUrl string    Git repository for chart templates (default "https://github.com/devspace-cloud/devspace-templates.git")
```

## File Structure
Running `devspace init` will create the following files for you:

```
YOUR_PROJECT_PATH/
|
|-- Dockerfile
|
|-- chart/
|   |-- Chart.yaml
|   |-- values.yaml
|   |-- templates/
|       |-- deployment.yaml
|
|-- .devspace/
|   |-- .gitignore
|   |-- config.yaml
```
