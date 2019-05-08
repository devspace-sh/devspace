---
title: Define config overrides
---

Using multiple configs is great for deploying to different environments but having multiple configs makes it a lot harder to keep everything consistent and up-to-date across different configurations.

To reduce the effort of maintaining many different configuration files with similar contents, DevSpace allows you to define config overrides for your configs. This allows yo to:

1. Create one config file (base config)
2. Define multiple configs in `devspace-configs.yaml` which all load the same config file (using the `path` option for each config)
3. Define config overrides for the different configs in `devspace-configs.yaml`

## Defining configs with overrides

```yaml
config1:
  config:
    path: ../devspace.yaml
config2:
  config:
    path: ../devspace.yaml
  overrides:
  - data:
      images:
        database:
          image: dscr.io/my-username/alternative-db-image
```
The above example defines two configurations `config1` and `config2`. Both will load the same config file `../devspace.yaml` but `config2` is slightly different because it will apply an override after loading the config file. This override defines that `images.database.image` should be overriden with the value `dscr.io/my-username/alternative-db-image`.

[Learn more about defining and using multiple configs.](/docs/configuration/multiple-configs)

## Advanced options for config overrides
Instead of specifying overrides directly inside `devspace-configs.yaml` with `data`, it is also possible to define a file containing the override data and reference this file with `path` instead of using `data`.

As shown in the example above, `overrides` is an array which allows you to apply multiple overrides. This can be useful when you want to re-use an override file multiple times but also apply additional overrides which are different between several configs.


---
## FAQ

<details>
<summary>
### Is it possible to override a single entry within an array? (e.g. overriding single deployments)
</summary>
**No.** It is, for example, not possible to override one specific deployment defined in the `deployments` section of a config file. Overriding the `deployments` would always override the entire array of `deployments`.
</details>

<details>
<summary>
### How is config overriding different from entrypoint overriding?
</summary>
Entrypoint overriding is a convenience feature that is specifically designed for `devspace dev` and will only be applied when running `devspace dev`. 

Config overriding is an advanced feature that allows you to override parts of a configuration. Config overrides impact any command you run, i.e. `devspace dev` AND `devspace deploy` as well as for any other command that you run using this overridden config.
</details>
