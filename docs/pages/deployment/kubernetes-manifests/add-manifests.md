---
title: Add Kubernetes manifests
---

If you already have existing Kubernetes manifests which you like to deploy using DevSpace CLI, you can easily add them to the `deployments` array defined in your `devspace.yaml` using the following command:
```bash
devspace add deployment [deployment-name] --manifests="./path/to/your/manifests/**"
```
Although you can add each manifest individually, you can also use the Glob format to add define a pattern of manifest paths that you want to deploy with DevSpace CLI. The above pattern would add all files within the `path/to/your/manifests` folder within the root directory of your project. Paths should be relative to the root directory of your project which also contains your `devspace.yaml`.

You can use [globtester.com](http://www.globtester.com/#p=eJzT0y9ILMnQL8nXr8wvLdLPTczLTEstLinW19ICAIcMCZc%3D&r=eJyVzMENgCAMAMBVDAPQBSq7VKiRhAKhlYTt9e3PAe4w5bnFQqq7E7J4ueChk11gDVa7BwjVfLKaQuJe2hKu5hdJwWMEhNcH%2FJEoj5kjf4YH8%2BAw7w%3D%3D&) to verify that your pattern matches the relative paths to your manifests.
