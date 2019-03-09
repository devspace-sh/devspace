---
title: Define dynamic configs
---

Dynamic configs let you define variables within your configuration. These variables will be filled with values that are stored outside of the config on the local machine of each developer. This allows DevOps engineers and team leaders to define a single configuration which can be versioned and distributed among all team members while still being able to set different options for different developers.

Additionally, dynamic configs can be very useful when defining secrets as environment variables in automation scenarios, e.g. when [using DevSpace within CI/CD pipelines](/docs/cli/deployment/pipelines).

> To be able to define dynamic configs, you need to be familiar with the basics of [using multiple configs](/docs/cli/configuration/multiple-configs).

## Define config variables
Config variables are placeholders with the format `${VAR_NAME}`. They have to be defined in the `vars` section of the `.devspace/configs.yaml`. You can then refer to these variables from within your configuration. You can use config variables within config files, within configs defined with `config.data` and within your overrides specified as `path` or `data`.
```
config1:
  config:
    path: .devspace/config.yaml
  overwrites:
  - data:
      images:
        database:
          image: ${ImageName}
  vars:
  - name: ImageName
    question: Which database image do you want to use?
```
If a user runs `devspace deploy` for the first time after defining the config variable as shown above, the question `Which database image do you want to use?` will appear within the terminal and the user would be asked to enter a value for this config variable. Setting a value for the config variables would **not** alter the configuration in any way because values of config variables are stored separately from the configuration.

> DevSpace CLI only asks the user once to provide the values for environment variables 

Currently, there is no convenience command for deleting the values of config variables. You can, however, remove config values manually from `.devspace/generated.yaml` if necessary.

## Using environment variables as config variables
The value for a config variable can also be set by defining an environment variable named `DEVSPACE_VAR_[VAR_NAME]`. Setting the value of a config variable with name `ImageName` would be possible by setting an environment value `DEVSPACE_VAR_IMAGENAME`.

<!--DOCUSAURUS_CODE_TABS-->
<!--Windows Powershell-->
```powershell
$env:DEVSPACE_VAR_IMAGENAME = "some-value"
```

<!--Mac Terminal-->
```bash
export DEVSPACE_VAR_IMAGENAME="some-value"
```

<!--Linux Bash-->
```bash
export DEVSPACE_VAR_IMAGENAME="some-value"
```
<!--END_DOCUSAURUS_CODE_TABS-->

Using environment variables to set dynamic configs can be particularly useful when defining secrets as environment variables in automation scenarios, e.g. when [using DevSpace within CI/CD pipelines](/docs/cli/deployment/pipelines).


---
## FAQ

<details>
<summary>
### What is the difference between config overrides and dynamic configs?
</summary>
While config overriding is very useful for different deployment environments, it is not as suitable for managing configs that vary between different developers. Imaging having 80 people working on a project: Do you want to manage 80 different configurations and version them via git? Dynamic configs allow you to version one config file which uses variables that are saved outside the git repository on the local machine of the developer.

Additionally, dynamic configs can be very useful when defining secrets as environment variables in automation scenarios, e.g. [using DevSpace within CI/CD pipelines](/docs/cli/deployment/pipelines).
</details>
