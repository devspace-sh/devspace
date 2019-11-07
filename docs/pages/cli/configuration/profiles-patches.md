---
title: Using Config Profiles & Config Patches
sidebar_label: Profiles & Patches
---

DevSpace allows you to define different profiles for different use cases (e.g. working on different services in the same project, starting certain debugging enviroment) or for different deployment targets (e.g. dev, staging production).

> Profiles allow you to modify the configuration by replacing entire sections of the config or by applying patches for certain parts of the base configuration.

A profile has to be configured in the `profiles` section of the `devspace.yaml`.
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/devbackend
      - image: john/debugger
profiles:
- name: production
  patches:
  - op: replace
    path: images.backend.image
    value: john/prodbackend
  - op: remove
    path: deployments[0].helm.values.containers[1]
  - op: add
    path: deployments[0].helm.values.containers
    value:
      image: john/cache
```

## Defining Profiles

### `profiles[*].name`
The `name` option expects a string defining the name of the profile.

> This option is mandatory when defining a profile.

#### Example: Setting Interactive Mode Images
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/devbackend
      - image: john/debugger
profiles:
- name: staging
  patches:
  - op: replace
    path: images.backend.image
    value: john/stagingbackend
  - op: remove
    path: deployments[0].helm.values.containers[1]
- name: production
  patches:
  - op: replace
    path: images.backend.image
    value: john/prodbackend
  - op: remove
    path: deployments[0].helm.values.containers[1]
  - op: add
    path: deployments[0].helm.values.containers
    value:
      image: john/cache
```
**Explanation:**  
- The above example defines 2 profiles: `staging`, `production`
- Users can use the flag `-p staging` to use the `staging` profile for a single command execution
- Users can use the flag `-p production` to use the `production` profile for a single command execution
- Users can permanently switch to the `staging` profile using: `devspace use profile staging`
- Users can permanently switch to the `production` profile using: `devspace use profile production`


### `profiles[*].patches`
The `patches` option expects a patch object which consists of the following properties:
- `op` stating the patch operation (possible values: `replace`, `add`, `remove`)
- `path` stating a jsonpath or an xpath within the config (e.g. `images.backend.image`, `deployments[0].helm.values.containers[1]`)
- `value` stating an arbirary value used by the operation (e.g. a string, an integer, a boolean, a yaml object)

> If you use the `replace` or `add` operation, `value` is a mandatory property.

> If `value` is defined, it must provide the correct type to be used when adding (`op = add`) or replacing (`op = replace`) the existing value found under `path` using the newly provided `value`.

#### Example: Setting Interactive Mode Images
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/devbackend
      - image: john/debugger
profiles:
- name: production
  patches:
  - op: replace
    path: images.backend.image
    value: john/prodbackend
  - op: remove
    path: deployments[0].helm.values.containers[1]
  - op: add
    path: deployments[0].helm.values.containers
    value:
      image: john/cache
```
**Explanation:**  
- The above example defines 1 profile: `production`
- When using the profile `production`, the config would be patched with 3 patches.
- The resulting config used in-memory when the profile `production` is used would look like this:

```yaml
# In-Memory Config After Applying Patches For Profile `production`
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/prodbackend
      - image: john/cache
```


### `profiles[*].replace`
The `patches` option expects a patch object which consists of the following properties:
- `op` stating the patch operation (possible values: `replace`, `add`, `remove`)
- `path` stating a jsonpath or an xpath within the config (e.g. `images.backend.image`, `deployments[0].helm.values.containers[1]`)
- `value` stating an arbirary value used by the operation (e.g. a string, an integer, a boolean, a yaml object)

> If you use the `replace` or `add` operation, `value` is a mandatory property.

> If `value` is defined, it must provide the correct type to be used when adding (`op = add`) or replacing (`op = replace`) the existing value found under `path` using the newly provided `value`.

#### Example: Setting Interactive Mode Images
```yaml
images:
  backend:
    image: john/devbackend
  backend-debugger:
    image: john/debugger
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/devbackend
      - image: john/debugger
profiles:
- name: production
  replace:
    images:
      backend:
        image: john/prodbackend
  patches:
  - op: replace
    path: images.backend.image
    value: john/prodbackend
  - op: remove
    path: deployments[0].helm.values.containers[1]
```
**Explanation:**  
- The above example defines 1 profile: `production`
- When using the profile `production`, the config section `images` would be entirely replaced and the additionally, 2 patches would be applied.
- The resulting config used in-memory when the profile `production` is used would look like this:

```yaml
# In-Memory Config After Applying Patches For Profile `production`
images:
  backend:
    image: john/prodbackend
deployments:
- name: app-backend
  helm:
    componentChart: true
    values:
      containers:
      - image: john/prodbackend
```

> As shown in this example, it is possible to use `replace` and `patch` options in combination when defining profiles.



<br>

---
## Useful Commands

### `devspace list profiles`
To get a list of available profiles, you can run this command:
```bash
devspace list profiles
```

### `devspace use profile [NAME]`
To permanently switch to a different profile, you can run this command:
```bash
devspace use profile [PROFILE_NAME]
```

> Permanently switching to a profile means that all future commands (e.g. `devspace deploy` or `devspace dev`) will be executed using this profile until the user [resets the profile setting](#devspace-use-profile-reset) (see below).

### `devspace use profile --reset`
To permanently switch back to the default configuration (no profile replaces and patches active), you can run this command:
```bash
devspace use profile --reset
```

### `devspace deploy|dev|... --profile=[NAME]`
Most DevSpace commands support the `-p / --profile` flag. Using this flag, you can run a single command with a different profile without switching your profile permenantly:
```bash
devspace build -p [PROFILE_NAME]
devspace deploy -p [PROFILE_NAME]
devspace dev -p [PROFILE_NAME]
devspace dev -i -p [PROFILE_NAME]
```
