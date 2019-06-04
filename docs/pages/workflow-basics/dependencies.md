---
title: Dependencies
---

DevSpace CLI allows you to define dependencies between several software projects, e.g. across different git repositories. This makes DevSpace CLI a great tool for building and deploying software that consists of several microservices.

When a DevSpace project has dependencies, DevSpace CLI will:
1. Resolve all dependencies in a resursive manner
2. Build a non-cyclic dependency tree
3. Choose a leave node from the dependency tree, build its images (unless skip is defined) and deploy its deployments
4. Remove the leave node from the tree and repeat step 3 until everything has been deployed

The algorithm used by DevSpace CLI for building and deploying dependencies ensures that all dependencies have been deployed in the correct order before the project you are calling DevSpace CLI from will be build and deployed.

## Define Dependencies
DevSpace CLI is able to work with dependencies from the following sources:
- `git`: defines a git repository as dependency (**recommended**)
- `path`: defines a dependency from a path relative to the current project's root directory

> Using `git` as dependency source is recommended because it makes it much easier to share the configuration with other developers on your team without forcing everyone to checkout the dependencies and placing them in the same folder structure.

### Define `git` Dependencies
Dependencies can be defined using the `dependencies` section within your `devspace.yaml`.
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
- source:
    git: https://github.com/my-auth-server 
```

The above example defines two dependencies using git repositories as source. DevSpace CLI will use your locally stored git credentials to clone the repositories into a temporary folder. Using the `devspace.yaml` within a dependency's repository, DevSpace CLI then builds the images defined and deploys the project's deployments.

### Define `path` Dependencies
If you want to define projects on your local machine as dependency, DevSpace CLI also supports `path` as dependency source.
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
- source:
    path: ../my-auth-server
  config: default
```
The above example would define one dependency using a git repository as source and a second dependency using a local path relative to the current project's root path.

> Using `path` source for dependencies is discouraged as it becomes an issue when sharing the configuration with other team members, i.e. using `path` dependencies requires everyone to clone all dependencies manually and use the same folder structure for all projects before using DevSpace CLI.

## Skip Image Building for Dependencies
It is very common that you wish to deploy a dependency but not to rebuild its images. For this case, DevSpace CLI allows you to set `skipBuild: true` as shown in the config example below:
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  skipBuild: true
```

## Ignore Dependencies of Dependencies
By default, DevSpace CLI resolves dependencies in a recursive manner. If only want to deploy the dependency itself, you can tell DevSpace CLI to ignore the dependency's child-dependencies by setting `ignoreDependencies: true` as shown in the example below:
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  ignoreDependencies: true
```

> Using `ignoreDependencies` can be useful to prevent problematic [circular dependencies](#circular-dependencies).

## Use Dependencies with Multiple Configs
If you define a dependency that has [multiple configs using a `devspace-configs.yaml`](/docs/configuration/multiple-configs), you can use the `config` option to define which config should be used to build and deploy this dependency.
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  config: staging
```
The above example would tell DevSpace CLI to use the config with name `staging` to build the dependencies images and deploy the deployments defines within this config of the dependency.

## Conflicts in Dependencies
DevSpace CLI know the following types of depenency conflicts:

### Redundant Dependencies
If DevSpace CLI detects that two projects within the dependency tree define the same child-dependency (i.e. a redundant dependency), DevSpace CLI will try to resolve this by removing the denepdency that is "higher" (i.e. found first when resolving dependencies) within the tree.

### Circular Dependencies
If DevSpace CLI two projects which define each other as dependencies (either directly or via child-dependencies), DevSpace CLI will terminate with an error showing the problematic dependency path within the dependency tree.

> To resolve circular dependencies, DevSpace CLI allows you to [ignore dependencies of dependencies](#ignore-dependencies-of-dependencies) by setting `ignoreDependencies: true` for a dependency.
