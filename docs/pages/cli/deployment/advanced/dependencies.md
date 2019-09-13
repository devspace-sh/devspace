---
title: Dependencies
---

DevSpace allows you to define dependencies between several software projects that have a `devspace.yaml`, e.g. across different git repositories. This makes DevSpace a great tool for building and deploying software that consists of several microservices.

## Dependency Resolution
When a DevSpace project has dependencies, DevSpace will:
1. Resolve all dependencies in a resursive manner
2. Build a non-cyclic dependency tree
3. Choose a leave node from the dependency tree, build its images (unless skip is defined) and deploy its deployments
4. Remove the leave node from the tree and repeat step 3 until everything has been deployed

The algorithm used by DevSpace for building and deploying dependencies ensures that all dependencies have been deployed in the correct order before the project you are calling DevSpace from will be built and deployed.

### Redundant Dependencies
If DevSpace detects that two projects within the dependency tree define the same child-dependency (i.e. a redundant dependency), DevSpace will try to resolve this by removing the denepdency that is "higher" (i.e. found first when resolving dependencies) within the tree.

### Circular Dependencies
If DevSpace two projects which define each other as dependencies (either directly or via child-dependencies), DevSpace will terminate with an error showing the problematic dependency path within the dependency tree.

> To resolve circular dependencies, DevSpace allows you to [ignore dependencies of dependencies](#ignore-dependencies-of-dependencies) by setting `ignoreDependencies: true` for a dependency.

<br>

---
## Dependency Source
DevSpace is able to work with dependencies from the following sources:
- `git`: defines a git repository as dependency that has a devspace configuration (**recommended**)
- `path`: defines a dependency from a local path relative to the current project's root directory

> Using `git` as dependency source is recommended because it makes it much easier to share the configuration with other developers on your team without forcing everyone to checkout the dependencies and placing them in the same folder structure.

### `dependencies[*].source.git`
The `source.git` option expects a string with the URL of a git repository. DevSpace will use the `master` branch by default and assumes that the `devspace.yaml` is located at the root directory of the git repository. To customize this behavior, use the following, complementary config options:
- [`branch` for a different git branch](#dependencies-sourcebranch)
- [`tag` for a specific git tag or release](#dependencies-sourcetag)
- [`revision` for a specific git commit hash](#dependencies-sourcerevision)
- [`subPath` for custom location of `devspace.yaml` within the repository](#dependencies-sourcesubpath)

> DevSpace will clone the git repository which is defined as a dependency and cache the project in the global cache folder (i.e. `$HOME/.devspace`). DevSpace will also pull new commits before deploying the dependency.

> **Authentication:** DevSpace is using the git credential store. So, if you are able to clone or pull from the specified repository, DevSpace will also be able to clone it or pull from there.

#### Example: Git Projects as Dependency
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
    branch: stable
- source:
    git: https://github.com/my-auth-server
    revision: c967392
- source:
    git: https://github.com/my-auth-server
    tag: v3.0.1
```
**Explanation:**  
- When you run `devspace deploy` or `devspace dev` for the first time after defining the dependencies, DevSpace will checkout all git projects into the global cache folder `$HOME/.devspace`.
- Wehenever you run `devspace deploy` or `devspace dev` (even the first time), DevSpace will:
  - Run a `git pull` for all cached repositories.
  - Load the `devspace.yaml` files of the dependency projects and resolve their dependencies respectively.
  - Deploy all dependency projects according to their `devspace.yaml` files.

### `dependencies[*].source.branch`

#### Default Value For `branch`
```yaml
branch: master
```

### `dependencies[*].source.tag`
### `dependencies[*].source.revision`
### `dependencies[*].source.subPath`


### `dependencies[*].source.path`
The `source.path` option expects a string with a relative path to a folder that contains a `devspace.yaml` which marks a project that is a dependency of the project referencing it.

> Using local projects with `path` option makes the configuration in `devspace.yaml` harder to share and is therefore discouraged.

#### Example: Local Project as Dependency
```yaml
dependencies:
- source:
    path: ../other-project
- source:
    path: ./different/subproject
```
**Explanation:**  
- Wehenever you run `devspace deploy` or `devspace dev`, DevSpace will:
  - Load the `devspace.yaml` files of both dependencies and resolve their dependencies respectively.
  - Deploy both projects according to their `devspace.yaml` files.


## Deployment Options

### `dependencies[*].profile`


#### Example: Use Config Profile for Dependency
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  profile: production
```

### `dependencies[*].skipBuild`
### `dependencies[*].ignoreDependencies`

> Using `ignoreDependencies` can be useful to prevent problematic [circular dependencies](#circular-dependencies).

### `dependencies[*].namespace`


<br>

---
## Useful Commands

### `devspace update dependencies`
#TODO

## OLD

### Define `git` Dependencies
Dependencies can be defined using the `dependencies` section within your `devspace.yaml`.
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
- source:
    git: https://myuser:mypass@my-private-git.com/my-auth-server 
```

The above example defines two dependencies using git repositories as source. DevSpace will use your locally stored git credentials to clone the repositories into a temporary folder. Using the `devspace.yaml` within a dependency's repository, DevSpace then builds the images defined and deploys the project's deployments.

### Define `path` Dependencies
If you want to define projects on your local machine as dependency, DevSpace also supports `path` as dependency source.
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
- source:
    path: ../my-auth-server
  config: default
```
The above example would define one dependency using a git repository as source and a second dependency using a local path relative to the current project's root path.

> Using `path` source for dependencies is discouraged as it becomes an issue when sharing the configuration with other team members, i.e. using `path` dependencies requires everyone to clone all dependencies manually and use the same folder structure for all projects before using DevSpace.

## Skip Image Building for Dependencies
It is very common that you wish to deploy a dependency but not to rebuild its images. For this case, DevSpace allows you to set `skipBuild: true` as shown in the config example below:
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  skipBuild: true
```

## Ignore Dependencies of Dependencies
By default, DevSpace resolves dependencies in a recursive manner. If only want to deploy the dependency itself, you can tell DevSpace to ignore the dependency's child-dependencies by setting `ignoreDependencies: true` as shown in the example below:
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  ignoreDependencies: true
```


## Use Dependencies with Multiple Configs
If you define a dependency that has [multiple configs using a `devspace-configs.yaml`](/docs/configuration/multiple-configs), you can use the `config` option to define which config should be used to build and deploy this dependency.
The above example would tell DevSpace to use the config with name `staging` to build the dependencies images and deploy the deployments defines within this config of the dependency.
