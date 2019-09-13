---
title: Shared Commands
sidebar_label: Shared Commands
id: version-v4.0.0-shared-commands
original_id: shared-commands
---

**Shared commands let you make rather complex commands easy to run for everyone on your team.**

The idea of shared commands is that the more experienced developers on your team define a set of useful commands and store them in the `devspace.yaml`, commit and push this config to the code repository via git and then, let other team mates run them without having to remember all the details or having to read through endless pages of internal workflow documentation.

## Workflow
The `commands` section in the `devspace.yaml` allows you to define commands that can be shared with other team mates. After adding a command and giving it a name (serving as an alias for the command), developers can run the command using:
```yaml
devspace run [command-name]
```

An example configuration could look like this:
```yaml
# File: devspace.yaml
images:
  default:
    image: john/backend
commands:
- name: debug-backend
  command: "devspace dev -i --profile=debug-backend"
profiles:
- name: debug-backend
  patches:
  - op: replace
    path: images.default.entrypoint
    value: ["npm", "run", "debug"]
```

> Shared commands do not have to use `devspace`, they can also run any other script or command, set environment variables etc. If you are familiar with the `scripts` section of the `package.json` for Node.js, you will find that `devspace run [name]` works pretty much the same way as `npm run [name]`

The above example configuration would allow a developer to the shared command `debug-backend` like this:
```bash
devspace run debug-backend
```

And `devspace run` would execute the following command internally:
```bash
devspace dev -i --profile=debug-backend
```

> Shared commands proxy input and output streams, so you can even share interactive commands such as `devspace enter`.

## Configuration

### `run[*].name`
The `name` option expects a string with name that serves as an alias for the command provided in the `command` option.

> The `name` option is mandatory and must be unique.

See above for an [example configuration](#workflow).


### `run[*].command`
The `command` option expects a string with an arbitrary terminal command. 

While you can run any `devspace` command, you can also run other commands (if installed), set environment variables or use `bash` style expressions such as `&&`, `||` or `;`. To ensure that many of your team mates can run the command on any platform, it is highly recommended to keep your command expressions as simple as possible.

> The `command` option is mandatory.

> Write all commands using `bash` style. DevSpace is using a library to make them as cross-platform executable as possible. 

See above for an [example configuration](#workflow).



## Useful Commands

### `devspace list commands`
To start development in interactive mode, run:
```bash
devspace list commands
```
