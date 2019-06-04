---
title: Switch between Spaces
---

# TODO @Fabian

The project you are currently running your commands in (i.e. you are in any sub-folder of your project within the terminal) has an active Space which is running all `devspace` commands with.

If you want to switch to another Space, simply run:
```bash
devspace use space [SPACE_NAME]
```
Possible use cases for this command would be:
1. You are using multiple Spaces for production, staging and development.
2. You got a new computer, cloned your project and want to re-connect your project to an already existing Space that you created on your old computer.

To get a list of all your Spaces, run:
```bash
devspace list spaces
```
