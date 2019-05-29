---
title: kaniko
---

If you wi

```yaml
images:
  default:
    image: dscr.io/username/image
    build:
      kaniko:
        cache: true
        flags: []
        insecure: false
        options:
          buildArgs:
            someArg: argValue
            anotherArg: anotherValue
```
