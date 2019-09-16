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
- Whenever you run `devspace deploy` or `devspace dev` (even the first time), DevSpace will:
  - Run a `git pull` for all cached repositories.
  - Load the `devspace.yaml` files of the dependency projects and resolve their dependencies respectively.
  - Deploy all dependency projects according to their `devspace.yaml` files.

### `dependencies[*].source.branch`
The `source.branch` option expects a string stating the branch of the git repository referenced via `source.git` that should be used when deploying this dependency.

#### Default Value For `source.branch`
```yaml
branch: master
```

### `dependencies[*].source.tag`
The `source.tag` option expects a string stating a tag of the git repository referenced via `source.git` that should be used when deploying this dependency.

### `dependencies[*].source.revision`
The `source.revision` option expects a string stating a commit hash of the git repository referenced via `source.git` that should be used when deploying this dependency.

### `dependencies[*].source.subPath`
The `source.subPath` option expects a string stating a folder within the git repository referenced via `source.git` that contains the `devspace.yaml` for this dependency.

#### Default Value For `source.subPath`
```yaml
subPath: /
```


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
- Whenever you run `devspace deploy` or `devspace dev`, DevSpace will:
  - Load the `devspace.yaml` files of both dependencies and resolve their dependencies respectively.
  - Deploy both projects according to their `devspace.yaml` files.


## Deployment Options
The following options allow you to customize the process used to deploy the dependency.

### `dependencies[*].profile`
The `profile` option expects a string with the name of a profile defined in the `devspace.yaml` of this dependency. When configuring this option, this profile will be used to deploy the dependency, i.e. the dependency will be deployed similar to running `devspace deploy -p [profile]` within the folder of the dependency.

#### Example: Use Config Profile for Dependency
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  profile: production
```


### `dependencies[*].skipBuild`
The `skipBuild` option expects a boolean with that defined if the image building process should be skipped when deploying this dependency. This is often useful if you rather want to use the tags that are defined in the deployment files (e.g. manifests or helm charts) which may reference more stable, production-like versions of the images.

> Using `skipBuild` is useful when trying to speed up the dependency deployment process, especially when working with many dependencies that frequently change.

#### Default Value For `skipBuild`
```yaml
skipBuild: false
```

#### Example: Skip Build for Dependency
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  skipBuild: true
```


### `dependencies[*].ignoreDependencies`
The `ignoreDependencies` option expects a boolean with that defined if the dependencies of this dependencies should not be resolved and deployed.

> Using `ignoreDependencies` can be useful to prevent problematic [circular dependencies](#circular-dependencies).

#### Default Value For `ignoreDependencies`
```yaml
ignoreDependencies: false
```

#### Example: Ignore Dependencies of Dependency
```yaml
dependencies:
- source:
    git: https://github.com/my-api-server
  ignoreDependencies: true
```

### `dependencies[*].namespace`
The `namespace` option expects a string stating a namespace that should be used to deploy the dependency to.

> By default, DevSpace deploys project dependencies in the same namespace as the project itself. 

> You should only use the `namespace` option if you are an advanced user because using this option requires any user that deploys this project to be able to create this namespace during the deployment process or to have access to the namespace with the current kube-context, if the namespace already exists.


<br>

---
## Useful Commands

### `devspace update dependencies`
If you want to force DevSpace to update the dependencies (e.g. git fetch & pull), you can run the following command:
```bash
devspace update dependencies
```
