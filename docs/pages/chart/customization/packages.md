---
title: Add helm packages
---

With DevSpace, it is very easy to add helm packages like databases and caches to your application.

## List all available packages
To show a list of available packages (Helm charts), run the following command:
```bash
devspace add package
```

## Add a package to your application
Adding a package (e.g. mysql) requires you to:
1. [Add the package as dependency of your Helm chart](#add-the-package-as-dependency)
2. [Re-deploy your application using `devspace deploy`](#re-deploy-your-application)

### Add the package as dependency
To add a package to the dependencies of your Helm chart, it is strongly recommended to use `devspace add package [PACKAGE_NAME]` instaed of manually adding dependencies to `chart/requirements.yaml`:
```bash
devspace add package mysql
```
The exemplary command above would:
1. Add `mysql` as dependency within `chart/requirements.yaml`
2. Add the most common config options to your `chart/values.yaml` under `mysql`
3. OPTIONAL: Add a selector for the package to `dev.selectors` in `.devspace/config.yaml` if supported for this package ([Learn more about selectors](/docs/cli/configuration/reference#devselectors))
4. OPTIONAL: Display the README of the `mysql` Helm chart

### Re-deploy your application
Adding a package as dependency will only change your local project files, e.g. `chart/requirements.yaml`. It does **not** automatically re-deploy your application with the new dependency. 

> After adding a package, you can customize the package configuration within the `chart/values.yaml` before deploying the newly added package.

When you are ready to update your deployed application, you can simply re-deploy your application with:
```
devspace deploy
```

## Remove a package
To a remove a previouly added package, you can simply run:
```
devspace remove package [PACKAGE_NAME]
devspace deploy
```

> If your packages created persistent volumes, you may have to delete them manually. [Learn more about deleting persistent volumes.](/docs/chart/customization/persistent-volumes#delete-persistent-volumes)

---
## FAQ

<details>
<summary>
### What are packages?
</summary>
Packages are Helm charts listed in the [stable repository within the official Helm/charts project on GitHub](https://github.com/helm/charts/tree/master/stable/). These charts often contain many best practices and allow for extensive configuration. Therefore, you should always check if anything your application needs as dependency is available as a package before adding it manually as a container within your chart.
</details>
