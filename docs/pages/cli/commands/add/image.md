---
title: devspace add image
sidebar_label: image
---

```bash
#######################################################
############# devspace add image ######################
#######################################################
Add a new image to your DevSpace configuration

Examples:
devspace add image my-image --image=dockeruser/devspaceimage2
devspace add image my-image --image=dockeruser/devspaceimage2 --tag=alpine
devspace add image my-image --image=dockeruser/devspaceimage2 --context=./context
devspace add image my-image --image=dockeruser/devspaceimage2 --dockerfile=./Dockerfile
devspace add image my-image --image=dockeruser/devspaceimage2 --buildengine=docker
devspace add image my-image --image=dockeruser/devspaceimage2 --buildengine=kaniko
#######################################################

Usage:
  devspace add image [flags]

Flags:
      --buildengine string   Specify which engine should build the file. Should match this regex: docker|kaniko
      --context string       The path of the images' context
      --dockerfile string    The path of the images' dockerfile
  -h, --help                 help for image
      --image string         The image name of the image (e.g. myusername/devspace)
      --tag string           The tag of the image
```
