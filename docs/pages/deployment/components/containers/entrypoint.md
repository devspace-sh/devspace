---
title: Entrypoint (cmd, args)
---

Components allow you to use the Kubernetes feature of overriding the container startup commands:
- `command` which will override the `ENTRYPOINT` specified in the Dockerfile
- `args` which will override the `CMD` specified in the Dockerfile

```yaml
deployments:
- name: backend
  component:
    containers:
    - image: dscr.io/username/api-server
      command:
      - sleep
      args:
      - 9999999
```
The above example would start the container effectively with the following command: `sleep 9999999`

For more information, please take a look at the [Kubernetes documentation for setting command and args](https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/).
