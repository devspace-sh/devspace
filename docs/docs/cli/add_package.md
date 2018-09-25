---
title: devspace add package
---

With `devspace add package`, you can easily add a package (helm chart) like mysql, nginx etc. to your devspace. To view all available packages run `devspace add package`.  

The devspace add package command adds the helm chart as a dependency in the requirements.yaml and calls the internal `helm dependency update`, which downloads the chart and places it in the chart/charts folder. To remove the dependency call `devspace remove package PACKAGE`.  

By default the standard stable helm chart repository is used (see: [Helm Charts](https://github.com/helm/charts/tree/master/stable)). If you want to add additional charts, just add the repository via `helm repo add` ([documentation](https://docs.helm.sh/helm/#helm-repo-add)).  

```
Usage:
  devspace add package [flags]

Flags:
      --app-version string     App version
      --chart-version string   Chart version
  -h, --help                   help for package
      --skip-question          Skips the question to show the readme in a browser

Examples:
devspace add package                                # Shows all available packages
devspace add package mysql                          # Adds the mysql chart to the devspace
devspace add package mysql --app-version=5.7.14     # Adds the mysql chart with app version 5.7.14 to the devspace
devspace add package mysql --chart-version=0.10.3   # Adds the mysql chart with chart version 0.10.3 to the devspace
```
