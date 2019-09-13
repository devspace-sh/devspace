---
title: Encryption keys
id: version-v3.5.18-access
original_id: access
---

When connecting a cluster to DevSpace Cloud, DevSpace CLI asks you to provide an encryption key. DevSpace CLI uses this key to encrypt the access to your Kubernetes cluster and send the encrypted access token to DevSpace Cloud. That means that DevSpace Cloud can only access your cluster when you provide the encryption key. DevSpace CLI securely stores your encryption key on your local machine and only sends it to DevSpace Cloud when running operations where DevSpace Cloud needs cluster access, e.g. when creating Spaces with `devspace create space [space-name]`.

## Resetting Encryption Keys

There are two methods how you can reset your cluster key, if you or another cluster user has forgotten his cluster key:
1. Reinvite the user through the UI. (Please be aware that you cannot reinvite yourself to a cluster)
2. If you are a cluster admin, you can also run `devspace reset key [cluster-name]` and select the admin kube context of the cluster where you want to reset the key for.
