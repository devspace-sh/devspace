---
title: Add Helm charts
---

DevSpace CLI lets you add existing Helm charts (either from your local filesystem or from a Helm registry) to your deployments.

## Add a local Helm chart
If you built your own Helm chart and it is located inside your project directory, you can simply add it as a deployment using the following command:
```bash
devspace add deployment [deployment-name] --chart="./path/to/my/chart"
```

## Add a Helm chart from a Helm repository
If you want to deploy a Helm chart from a chart repository, you can simply add it as shown in this example:
```bash
devspace add deployment [deployment-name] --chart="stable/mysql"
```

You can replace `stable` with the name of your Helm chart repository, if the repository already exists on your local computer. If you want to use a chart from a chart repository that you have not used yet, you can also specify the full repository URL:
```bash
devspace add deployment [deployment-name] --chart="chart-name" --chart-repo="https://my-chart-repository.tld"
```

> Use the `--chart-version` flag to specifiy the char version that you want to deploy.
