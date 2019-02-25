---
title: Variables & Dynamic Configuration
---

# Variables
With devspace it is possible to make parts of the configuration dynamic:

You can substitute every value in the configuration to load from either an environment variable or prompt the user to enter it. Take this simple configuration that just deploys a simple helm chart as example:

```
version: v1alpha1
devspace:
  deployments:
  - name:
      fromVar: DeploymentName
    helm:
      chartPath: ./chart
```

If you run `devspace up`, devspace will prompt you for the value of variable `DeploymentName` before beginning to deploy the helm chart. You can also customize what question, default value and validation pattern is used in order to prompt the value.

Now run `devspace reset vars` to 
