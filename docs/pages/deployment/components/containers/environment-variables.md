---
title: Environment variables
---

Instead of storing configuration data (e.g. database host, username and password) inside your Docker image, you should define such information as environment variables within your Helm chart.

## Setting environment variables
You can define environment variables for your containers in the `components[*].container[*].env` section within `chart/values.yaml`.
```yaml
components:
- name: default
  containers:
  - image: "dscr.io/username/mysql"
    env:
    - name: MYSQL_USER
      value: "my_user"
    - name: MYSQL_PASSWORD
      value: "my-secret-passwd"
```
The above example would set two environment variables, `MYSQL_USER="my_user"` and `MYSQL_PASSWORD="my-secret-passwd"` within the first container of the `default` component.

<details>
<summary>
### View the specification for environment variables
</summary>
```yaml
name: [a-z0-9-]{1,253}      # Name of the environment variable
values: [string]            # Option 1: Set static value for the environment variable
valueFrom:                  # Option 2: Load value from another resource
  secretKeyRef:             # Option 2.1: Use the content of a Kubernetes secret as value
    name: [secret-name]     # Name of the secret
    key: [key-name]         # Key within the secret
  configMapKeyRef:          # Option 2.2: Use the content of a Kubernetes configMap as value
    name: [configmap-name]  # Name of the config map
    key: [key-name]         # Key within the config map
```
The value of an environment variable can be either set:
1. By directly inserting the value via `value`
2. By referencing a key within a secret via `valueFrom.secretKeyRef`
3. By referencing a key within a configMap via `valueFrom.configMapKeyRef`
4. By using any other field supported for `valueFrom` as defined by the [Kubernetes specification for `v1.EnvVarSource`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#envvarsource-v1-core)
</details>
