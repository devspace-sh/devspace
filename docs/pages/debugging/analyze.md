---
title: Analyze issues
---

## Analyze your Space to identify issues
DevSpace.cli can automatically analyze your Space and identify potential issues with your deployments.
```bash
devspace use space [SPACE_NAME]
devspace analyze
```
Running `devspace analyze` will show a lot of useful debugging information, including:
- Containers that are not starting due to failed image pulling
- Containers that are not starting due to issues with the entrypoint command
- Network issues related to unhealthy pods and missing endpoints for services

<details>
<summary>
### Show an exemplary output of `devspace analyze`
</summary>
```bash
TODO
```
</details>
