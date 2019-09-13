---
title: Using DevSpace Cloud
id: version-v3.5.18-access
original_id: access
---

There are two ways how you can access DevSpace Cloud:
1. Use the public instance under https://app.devspace.cloud
2. Install DevSpace Cloud in your own cluster. For more informations how to install your own DevSpace Cloud instance, see the [official repository](https://github.com/devspace-cloud/devspace-cloud)

## Configure DevSpace CLI to access your own DevSpace Cloud instance

DevSpace CLI by default uses the official DevSpace Cloud instance under https://app.devspace.cloud. If you want to change the default DevSpace Cloud instance, you have to run the following command in your terminal:

```bash
# Add your provider to the provider list
devspace add provider devspace.my-domain.com

# Set default provider to your DevSpace Cloud instance
devspace use provider devspace.my-domain.com
```

This will tell DevSpace CLI to use this instance from now on for [connecting clusters](https://devspace.cloud/docs/cloud/clusters/connect), [creating spaces](https://devspace.cloud/docs/cloud/spaces/create-spaces) and opening the UI. If you have deployed a private docker registry with DevSpace Cloud, DevSpace CLI will also login to that registry automatically if you add that provider.
