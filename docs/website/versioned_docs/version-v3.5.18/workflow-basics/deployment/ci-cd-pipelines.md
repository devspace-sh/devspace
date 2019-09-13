---
title: CI/CD Integration
id: version-v3.5.18-ci-cd-pipelines
original_id: ci-cd-pipelines
---

DevSpace CLI is able to run in non-interactive environments such as in CI/CD pipelines. This allows you to use the DevSpace configuration not only for development but also for spinning up test environments and updating your staging or production system with automated deployment scripts that use DevSpace CLI.

## Login with an Access Key
To use DevSpace Cloud in CI/CD pipelines, you need to use access keys for authentication. To create a new access key navigate to `Settings -> Access Keys` in the DevSpace Cloud UI and click on the `Create Key` button. Follow the instructions on the screen and copy the newly created access key.  

In your CI/CD pipeline, you can now login into your account with: 
```bash
devspace login --key=ACCESS_KEY
```

After running the above command for authentication with an access key, you can use the usual DevSpace commands within your CI/CD pipeline, e.g. `devspace create space`, `devspace use space` and `devspace remove space`.  
