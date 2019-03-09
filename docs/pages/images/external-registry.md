---
title: Use external registries
---

DevSpace CLI works with public or private Docker registry. The only thing you need to do is to login to those registries with the `docker login` command.

## Use Docker Hub
To use Docker Hub with DevSpace CLI, login to Docker Hub first:
```bash
docker login
```
Afterwards, you can use images of this format: `[DOCKER_HUB_USERNAME]/[IMAGE]:[TAG]`

## Use your another external Docker registry
To use another hosted registry or even your own private Docker registry with DevSpace CLI, login to the registry first:
```bash
docker login [registry.tld]
```
Afterwards, you can use images of this format: `[registry.tld]/[USERNAME]/[IMAGE]:[TAG]`
