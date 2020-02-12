---
title: Dynamic Configuration using Config Variables
sidebar_label: Config Variables
id: version-v4.0.0-variables
original_id: variables
---

DevSpace allows you to make your configuration dynamic by using variables in `devspace.yaml`.

> DevSpace allows you to use environment variables without explicitly defining them as varaiables. You can simply reference them via `${MY_ENV_VAR}` anywhere in your `devspace.yaml`.

> DevSpace provides a set of [predefined variables](#predefined-variables) that are prefixed with `DEVSPACE_`.

While there is no need to explicitly define a config variable, it allows you to customize the behavior of DevSpace when working with the variable. Variables are defined within the `vars` section of `devspace.yaml`.
```yaml
vars:
- name: IMAGE_NAME
  question: Which database image do you want to use?
- name: HOME
  source: env
- name: REGISTRY
  question: Which registry do you want to push to?
  source: input
  options:
  - hub.docker.com
  - my.private-registry.tld
  - dscr.io
```

## Define Config Variables

### `name`
The `name` option expects a string stating the name of the config variable that will be used to reference it within the remainder of the configuration.

> The `name` of a config variable must be unique and is mandatory when defining a config variable.

> If the user will be asked to provide a value, the value will be cached in `.devspace/generated.yaml`.


### `source`
The `source` option expects either:
- `all` means to check environment variables **first** and then check for cached values in `.devspace/generated.yaml` (**default**)
- `env` means to check environment variables only
- `input` means to check user-provided, cached values in `.devspace/generated.yaml`

> If `source` is either `all` or `input` and the variable is not defined, the user will be asked to provide a value either using a generic question or the one provided via the [`question` option](#question). The user-provided value will be cached in `.devspace/generated.yaml`.

> If the `source` is `env`, DevSpace will always prefer to use the value of the environment variable, even if there is a cached value for the variable in `.devspace/generated.yaml`.

> Is the `source` is `env` and the environment variable is **not** defined, DevSpace will use the [`default` value](#default) or terminate with a fatal error, if there is **no** [`default` value](#default) configured.

#### Default Value For `source`
```yaml
source: all
```


### `default`
The `default` option expects a string defining the default value for the variable.

> Is the [`source`](#source) is `env` and the environment variable is **not** defined, DevSpace will use the `default` value or terminate with a fatal error, if there is **no** `default` value configured.


### `question`
The `question` option expects a string with a question that will be asked when the variable is not defined. DevSpace tries to resolve the variable according to the [`source` of the variable](#source) and if it is not set via any of the accepted sources, DevSpace will prompt the user to define the value by entering a string.

> Defining the `question` is optional but often helpful to provide a better usability for other team members using the project.

> If [valid `options` for the variable value](#options) are configured, DevSpace will show a picker/selector instead of a regular input field/prompt.

> If a [`default` value](#default) is configured for the variable, DevSpace will use this [`default` value](#default) as default answer for the question that can be easily selected by pressing enter.

#### Default Value For `question`
```yaml
question: Please enter a value for [VAR_NAME] # using the variable name
```


### `options`
The `options` option expects an array of strings with each string stating a allowed value for the variable.

#### Example: Define Variable Options
```yaml
vars:
- name: REGISTRY
  question: Which registry do you want to push to?
  source: input
  options:
  - hub.docker.com
  - my.private-registry.tld
  - dscr.io
  default: my.private-registry.tld
```
**Explanation:**  
If the variable REGISTRY is used for the first time during `devspace deploy`, DevSpace will ask the user to select which value to use by showing this question:
```bash
Which registry do you want to push to? (Default: my.private-registry.tld)
Use the arrows UP/DOWN to select an option and ENTER to choose the selected option.
  hub.docker.com
> my.private-registry.tld
  dscr.io
```

### `validationPattern`
The `validationPattern` option expects a string stating a regular expression that validates if the value entered by the user is allowed as a value for this variable.

> If the provided value does not match the regex in `validationPattern`, DevSpace will either show a generic error message or the message provided in [`validationMessage`](#validationmessage).


### `validationMessage`
The `validationMessage` option expects a string stating an error message that is shown to the user when providing a value for the variable that does not match the regex provided in [`validationPattern`](#validationpattern).


<br>

---
## Predefined Variables

DevSpace provides some variables that are filled automatically and can be used within the config. These can be helpful for image tagging and other use cases:

- **DEVSPACE_RANDOM**: A random 6 character long string
- **DEVSPACE_TIMESTAMP** A unix timestamp when the config was loaded
- **DEVSPACE_GIT_COMMIT**: A short hash of the local repos current git commit
- **DEVSPACE_SPACE**: The name of the [space](../../cloud/spaces/what-are-spaces) that is currently used
- **DEVSPACE_SPACE_NAMESPACE**: The kubernetes namespace of the [space](../../cloud/spaces/what-are-spaces) in the cluster
- **DEVSPACE_SPACE_DOMAIN1**, **DEVSPACE_SPACE_DOMAIN2**... : The connected domains of the [space](../../cloud/spaces/what-are-spaces). E.g. if a space has a domain connected with test.devspace.host, **DEVSPACE_SPACE_DOMAIN1** will hold test.devspace.host
- **DEVSPACE_USERNAME**: The username currently logged into devspace cloud

### Example: Using `${DEVSPACE_GIT_COMMIT}`
```yaml
images:
  default:
    image: myrepo/devspace
    tag: ${DEVSPACE_GIT_COMMIT}-${DEVSPACE_TIMESTAMP}
```
**Explanation:**  
This config will tag the image in the form of `myrepo/devspace:d9b4bcd-1559766514`. Many other combinations are possible with this method.

<br>

---
## Useful Commands

### `devspace list vars`
To get a list of all variables defined in the `devspace.yaml`, you can run this command:
```bash
devspace list vars
```

### `devspace reset vars`
Once DevSpace asks you to provide a value for a variable, this value will be stored in the variables cache, so you will not asked about this variable again.

To reset the variables cache:
```bash
devspace reset vars
```

> DevSpace will fill the variables cache again, once you run the next build or deployment command.


### `export VAR_NAME=value`
The value for a config variable can also be set by defining an environment variable named `[VAR_NAME]`. Setting the value of a config variable with name `${IMAGE_NAME}` would be possible by setting an environment value `IMAGE_NAME`.

<!--DOCUSAURUS_CODE_TABS-->
<!--Windows Powershell-->
```powershell
$env:IMAGE_NAME = "some-value"
```

<!--Mac Terminal-->
```bash
export IMAGE_NAME="some-value"
```

<!--Linux Bash-->
```bash
export IMAGE_NAME="some-value"
```
<!--END_DOCUSAURUS_CODE_TABS-->
