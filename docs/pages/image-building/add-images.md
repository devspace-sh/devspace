---
title: Add images
---


To tell DevSpace CLI to build an additional image, simply use the `devspace add image` command.
```bash
devspace add image database --dockerfile=./db/Dockerfile --context=./db --image=dscr.io/username/mysql
```

The command shown above would add a new image to your DevSpace configuration. The resulting configuration would look similar to this one:

```yaml
images:
  database:                         # from --name
    image: dscr.io/username/image   # from args[0]
    dockerfile: ./db/Dockerfile     # from --dockerfile
    context: ./db                   # from --context
```
