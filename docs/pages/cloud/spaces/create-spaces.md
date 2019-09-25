---
title: Create Spaces
---

There are two ways how a [Space](../../cloud/spaces/what-are-spaces) can be created:
1. By the cli via `devspace create space` or the UI
2. By the cluster admin via `Clusters -> Click On Cluser -> Spaces -> Create Space`

Your Space name should only contain the following characters: `a-z`, `0-9`, `-` and space names must be unique for each user (but two users can have a space with the same name). 

> Creating a Space will automatically switch the `kubectl` context and use it for the current project (if there is any).

## Accessing Spaces

To configure your local kube context or project to use a certain space just run the following command anywhere on your computer (or in the project you want to use the space for):

```bash
devspace use space NAME
```

> This will also change the current `kubectl` context to the space context.  

To get a list of all your Spaces, run:
```bash
devspace list spaces
```

If you are a cluster admin you can also access the spaces of other users. Run:
```bash
devspace list spaces --all
```
to also display spaces from other users. Spaces from other users are displayed in the form of `username:spacename` and can be used via `devspace use space username:spacename`.

## Create a Space for another cluster user

Navigate to `Clusters -> Click On Cluser -> Spaces`. Click on the `Create Space` button and select an user you want to create a space for. You are now able to specify the space limits for this new space. The user is then able to access the space via `devspace use space NAME`. 

## Switch between Spaces

When using a command like `devspace use space` or `devspace create space` within a folder that has a `devspace.yaml`, DevSpace will switch the currently used space and write it to the `.devspace/generated.yaml`. So the next time you run `devspace deploy` or `devspace dev` the previously configured space will be used.  

If you want to stop using a space for a porject and switch to the currently active kubectl context you can run `devspace use space none`. 

If you want to switch to another Space, simply run:
```bash
devspace use space [SPACE_NAME]
```

Possible use cases for this command would be:
1. You are using multiple Spaces for production, staging and development.
2. You got a new computer, cloned your project and want to re-connect your project to an already existing Space that you created on your old computer.
