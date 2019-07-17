---
title: Dynamic config variables
---

DevSpace allows you to write dynamic configs by defining variables within your configuration. These variables will be filled with values that are stored outside of the config on the local machine of each developer or are entered by the developer. DevSpace CLI also provides some predefined variables that can be used and are filled automatically during deployment. This allows DevOps engineers and team leaders to define a single configuration which can be versioned and distributed among all team members while still being able to set different options for each developer.  

Additionally, dynamic configs can be very useful when defining secrets as environment variables in automation scenarios, e.g. when using DevSpace within CI/CD pipelines.

> To be able to define dynamic configs, you need to be familiar with the basics of [using multiple configs](/docs/configuration/multiple-configs).

## Define config variables
Config variables are placeholders with the format `${VAR_NAME}` and can be used for ANY value within the devspace config. They have to be defined in the `vars` section of the `devspace-configs.yaml`. You can then refer to these variables from within your configuration. You can use config variables within config files, within configs defined with `config.data` and within your overrides specified as `path` or `data`.
```
config1:
  config:
    path: ../devspace.yaml
  overrides:
  - data:
      images:
        database:
          image: ${ImageName}
  vars:
  - name: ImageName
    question: Which database image do you want to use?
```

If a user runs `devspace deploy` for the first time after defining the config variable as shown above, the question `Which database image do you want to use?` will appear within the terminal and the user would be asked to enter a value for this config variable. Setting a value for the config variables would **not** alter the configuration in any way because values of config variables are stored separately from the configuration in the `.devspace/generated.yaml`. You can print all configured variables with `devspace list vars`.  

> DevSpace CLI only asks the user once to provide the values for environment variables 

> For a working example take a look at [dynamic-config](https://github.com/devspace-cloud/devspace/tree/master/examples/dynamic-config)

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

Using environment variables to set dynamic configs can be particularly useful when defining secrets as environment variables in automation scenarios, e.g. when using DevSpace within CI/CD pipelines.

## Predefined Variables

DevSpace provides some variables that are filled automatically and can be used within the config. These can be helpful for image tagging and other use cases:

- **DEVSPACE_RANDOM**: A random 6 character long string
- **DEVSPACE_TIMESTAMP** A unix timestamp when the config was loaded
- **DEVSPACE_GIT_COMMIT**: A short hash of the local repos current git commit
- **DEVSPACE_SPACE**: The name of the [space](/docs/cloud/spaces/what-are-spaces) that is currently used
- **DEVSPACE_USERNAME**: The username currently logged into devspace cloud

For example these predefined variables can be used to dynamically tag images during deployment:

```yaml
images:
  default:
    image: myrepo/devspace
    tag: ${DEVSPACE_GIT_COMMIT}-${DEVSPACE_TIMESTAMP}
```

This config will tag the image in the form of `myrepo/devspace:d9b4bcd-1559766514`. Many other combinations are possible with this method.

## Variable Reference

### config.vars[\*]
```yaml
vars:                               # struct   | Options for variables
- name: ""                          # string   | The name of the variable (can be used within the config as ${name}) and can be defined via environment variable as DEVSPACE_VAR_NAME
  question: "How do you ..."        # string   | Question that will be presented to the user for filling the value
  options: []                       # string[] | Array of possible answer options for the variable value
  default: ""                       # string   | Default value of the variable if user skips question
  validationPattern: "^.*$"         # string   | Regex pattern to verify the variable input
  validationMessage: "Wrong ..."    # string   | The error message to print if the entered value does not match the pattern
```

---
## FAQ

<details>
<summary>
### What is the difference between config overrides and dynamic configs?
</summary>
While config overriding is very useful for different deployment environments, it is not as suitable for managing configs that vary between different developers. Imaging having 80 people working on a project: Do you want to manage 80 different configurations and version them via git? Dynamic configs allow you to version one config file which uses variables that are saved outside the git repository on the local machine of the developer.

Additionally, dynamic configs can be very useful when defining secrets as environment variables in automation scenarios, e.g. using DevSpace within CI/CD pipelines.
</details>
