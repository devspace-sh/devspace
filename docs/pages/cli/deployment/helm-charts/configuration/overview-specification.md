---
title: Configure Helm Chart Deployments
sidebar_label: Helm Charts
---

To deploy Helm charts, you need to configure them within the `deployments` section of the `devspace.yaml`.
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    values:
      mysqlRootPassword: ${MYSQL_ROOT_PASSWORD}
      mysqlUser: db_user
      mysqlDatabase: app_database
- name: backend
  helm:
    chart:
      name: backend-chart
      repository: https://my-repo.tld/
```

[What are Helm charts?](../../../../cli/deployment/helm-charts/what-are-helm-charts)

## Chart

### `deployments[*].helm.chart.name`
The `name` option expects a string stating either:
- a path to a chart that is stored on the filesystem
- the name of a chart that is located in a repository (either the default repository or one specified via [`repo` option](#deployments-helmchartrepo))

DevSpace follows the same behavior as `helm install` and first checks if the path specified in `name` exists on the file system and is a valid chart. If not, DevSpace will assume that the `name` is not a path but the name of a remote chart located in a chart repository.

> Specifying the `name` option for Helm deployments is mandatory.

#### Example: Simple Helm Deployment
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql
```

### `deployments[*].helm.chart.version`
The `version` option expects a string stating the version of the chart that should be used.

> If no version is specified, Helm would try to get the latest version of this chart.

#### Default Value for `version`
```yaml
version: ""
```

#### Example: Custom Chart Version
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
      version: "1.3.1"
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql --version="1.3.1"
```

### `deployments[*].helm.chart.repo`
The `repo` option expects a string with an URL to a [Helm Chart Repository](https://helm.sh/docs/chart_repository/).

> The [official Helm Chart Repository `stable`](https://github.com/helm/charts) does not need to be specified as serves as default value.

#### Default Value for `repo`
```yaml
repo: stable
```

#### Example: Custom Chart Repository
```yaml
deployments:
- name: database
  helm:
    chart:
      name: custom-chart
      repository: https://my-repo.tld/
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database custom-chart --repo "https://my-repo.tld/"
```

## Values Overriding

### `deployments[*].helm.values`
The `values` option expects an object with values that should be overriding the default values of this Helm chart.

Compared to the `valuesFiles` option, using `values` has the following advantages:
- It is easier to comprehend and faster to find (no references)
- It allows you to use [dynamic config variables](../../../../cli/configuration/variables)

> Because both, `values` and `valuesFiles`, have advantages and disadvantages, it if often useful to combine them. When setting both, values defined in `values` have precedence over values defined in `valuesFiles`.

#### Default Value for `values`
```yaml
values: {}
```

#### Example: Using Values in devspace.yaml
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    values:
      mysqlRootPassword: ${MYSQL_ROOT_PASSWORD}
      mysqlUser: db_user
      mysqlDatabase: app_database
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql --set mysqlRootPassword="$MYSQL_ROOT_PASSWORD" --set mysqlUser="db_user" --set mysqlDatabase="app_database"
```

### `deployments[*].helm.valuesFiles`
The `valuesFiles` option expects an array of paths to yaml files which specify values for overriding the values.yaml of the Helm chart.

Compared to the `values` option, using `valuesFiles` has the following advantages:
- It reduces the size of your `devspace.yaml` especially when setting many values for a chart
- It allows you to run Helm commands directly without DevSpace, e.g. `helm upgrade [NAME] -f mysql/values.yaml`

> Because both, `values` and `valuesFiles`, have advantages and disadvantages, it if often useful to combine them. When setting both, values defined in `values` have precedence over values defined in `valuesFiles`.

#### Default Value for `valuesFiles`
```yaml
valuesFiles: []
```

#### Example: Using Values Files
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    valuesFiles:
    - mysql/values.yaml
    - mysql/values.production.yaml
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql -f mysql/values.yaml -f mysql/values.production.yaml
```


### `deployments[*].helm.replaceImageTags`
The `replaceImageTags` option expects a boolean stating if DevSpace should do [Image Tag Replacement](../../../../cli/deployment/workflow-basics#3-tag-replacement).

By default, DevSpace searches all your values (specified via `values` or `valuesFiles`) for images that are defined in the `images` section of the `devspace.yaml`. If DevSpace finds an image, it replaces or appends the image tag with the tag it created during the [image building process](../../../../cli/image-building/workflow-basics). Image tag replacement makes sure that your application will always be started with the most up-to-date image that DevSpace has built for you.

> Tag replacement takes place **in-memory** and is **not** writing anything to the filesystem, i.e. it will **never** change any of your configuration files.

#### Default Value for `replaceImageTags`
```yaml
replaceImageTags: true
```

#### Example: Disable Tag Replacement
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    replaceImageTags: false
```



## Helm Options

### `deployments[*].helm.wait`
The `wait` option expects a boolean that will be used for the [helm flag `--wait`](https://helm.sh/docs/using_helm/#helpful-options-for-install-upgrade-rollback).

#### Default Value for `wait`
```yaml
wait: false
```

#### Example: Helm Flag Wait
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    wait: true
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql --wait
```

### `deployments[*].helm.timeout`
The `timeout` option expects an integer representing a number of seconds that will be used for the [helm flag `--timeout`](https://helm.sh/docs/using_helm/#helpful-options-for-install-upgrade-rollback).

#### Default Value for `timeout`
```yaml
timeout: 180
```

#### Example: Helm Flag Timeout
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    timeout: 300
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql --timeout=300
```

### `deployments[*].helm.force`
The `force` option expects a boolean that will be used for the [helm flag `--force`](https://helm.sh/docs/helm/#helm-upgrade).

#### Default Value for `force`
```yaml
force: false
```

#### Example: Helm Flag Force
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    force: true
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql --force
```

### `deployments[*].helm.rollback`
The `rollback` option expects a boolean that states if DevSpace should automatically rollback deployments that fail.

#### Default Value for `rollback`
```yaml
rollback: false
```

#### Example: Enabling Automatic Rollback
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    rollback: true
```

### `deployments[*].helm.tillerNamespace`
The `tillerNamespace` option expects a string that will be used for the [helm flag `--tiller-namespace`](https://helm.sh/docs/using_helm/#helpful-options-for-install-upgrade-rollback).

#### Default Value for `tillerNamespace`
```yaml
tillerNamespace: "" # defaults to default namespace of current context
```

#### Example: Helm Flag Force
```yaml
deployments:
- name: database
  helm:
    chart:
      name: stable/mysql
    tillerNamespace: my-tiller-ns
```
**Explanation:**  
Deploying the above example would roughly be equivalent to this command:
```bash
helm install --name database stable/mysql --tiller-namespace=my-tiller-ns
```



<br>

---
## Useful Commands

### `devspace add deployment [NAME] --chart="./path/to/my/chart"`
If you built your own Helm chart and it is located inside your project directory, you can simply add it as a deployment using the following command:
```bash
devspace add deployment [deployment-name] --chart="./path/to/my/chart"
```

> Running `devspace add deployment` only adds a deployment to `devspace.yaml` but does not actually deploy anything. To deploy the newly added deployment, run `devspace deploy` or `devspace dev`.

### `devspace add deployment [deployment-name] --chart="stable/[CHART]"`
If you want to deploy a Helm chart from a chart repository, you can simply add it as shown in this example:
```bash
devspace add deployment [deployment-name] --chart="stable/mysql"
```
You can replace `stable` with the name of your Helm chart repository, if it already exists on your local computer. If you want to use a chart from a chart repository that you have not used yet, you can also specify the repository URL:
```bash
devspace add deployment [deployment-name] --chart="chart-name" --chart-repo="https://my-chart-repository.tld"
```
Use the `--chart-version` flag to specifiy the char version that you want to deploy.

> Running `devspace add deployment` only adds a deployment to `devspace.yaml` but does not actually deploy anything. To deploy the newly added deployment, run `devspace deploy` or `devspace dev`.


### `devspace remove deployment [NAME]`
Instead of manually removing a deployment from your `devspace.yaml`, it is recommended to run this command instead:
```bash
devspace remove deployment [deployment-name]
```

The benefit of running `devspace remove deployment` is that DevSpace will ask you this question:
```bash
? Do you want to delete all deployment resources deployed?  [Use arrows to move, type to filter]
> yes
  no
```

If you select yes, DevSpace  will remove your deployment from your Kubernetes cluster before deleting it in your `devspace.yaml`. This is great to keep your Kubernetes namespaces clean from zombie deployments that cannot be easily tracked, removed and updated anymore.
