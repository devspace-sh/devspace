---
title: Using Multiple Configurations
sidebar_label: Multiple Configs
---

Sometimes it might be useful to define multiple configurations (e.g. for deploying to different environments). To support this case, DevSpace CLI allows you to create the file `devspace-configs.yaml` where you can define multiple configurations.

> Using multiple configs is an advanced feature. To define a different behavior for `devspace deploy` and `devspace dev`, you should consider [overriding the entrypoints of your images](/docs/development/overrides#configuring-entrypoint-overrides).

## Defining multiple configurations
Multiple configurations can be defined in `devspace-configs.yaml`.
```yaml
config1:
  config:
    path: ../devspace.yaml
config2:
  config:
    data:
      version: v1alpha2
      dev: ...
      deployments: ...
      images: ...
```
A config can either be loaded from a `path` or it can be defined directly inside `devspace-configs.yaml` using the `data` key.

The above example defines two configurations. The first one is called `config1` and will be loaded from the path `../devspace.yaml`. The second one is called `config2` and is directly defined within the `data` section in this `devspace-configs.yaml` file.

> Instead of creating multiple completely different configuration files, it is often much better to use [config overrides](/docs/configuration/overrides) which allow you to have multiple slightly different configurations on top of a single configuration file.

## Switching between multiple configs
To switch between different configs, you can run:
```bash
devspace use config [CONFIG_NAME]
```

> After adding a newly created `devspace-configs.yaml` to your project, you will need to run `devspace use config [CONFIG_NAME]` to tell DevSpace CLI which configuration to use.

## List all configs
To get a list of defined config, you can run:
```bash
devspace list configs
```
