---
title: How To Configure Differences Between Dev, Staging & Production
sidebar_label: Dev vs. Staging vs. Production
id: version-v4.2.0-development-staging-production
original_id: development-staging-production
---

To configure the differences between development, staging and production environments, there are several techniques with may be used separtely or combined in certain use cases:
- For image building:
    1. Using [different Dockerfiles](../../cli/image-building/configuration/overview-specification#images-dockerfile) for each environment
    2. Using the same Dockerfile and:
       - [overriding the `ENTRYPOINT` or the `CMD` of the Dockerfile](../../cli/image-building/configuration/overview-specification#overriding-entrypoint-amp-cmd) for each environment
       - using multi-stage builds and [setting the `target`](../../cli/image-building/configuration/build-options#target) during the build process
       - using [different `buildArgs`](../../cli/image-building/configuration/build-options#buildargs) for different environments
- For deployments:
    1. Using a different `cmd` and/or `args` for your containers depending on the environment
    2. Using a different `image` name or `image` tag for your containers depending on the environment
    3. Setting different `env` variables for your containers depending on the environment

No matter which options are working for your use case, the following DevSpace features will allow you to set up the desired differences:
- [Config Profiles &amp; Patches](../../cli/configuration/profiles-patches)
- [Config Variables](../../cli/configuration/variables)
- [Hooks](../../cli/configuration/hooks)
