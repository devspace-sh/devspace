---
title: Encryption keys
---

When connecting a cluster to DevSpace Cloud, DevSpace CLI asks you to provide an encryption key. DevSpace CLI uses this key to encrypt the access to your Kubernetes cluster and send the encrypted access token to DevSpace Cloud. That means that DevSpace Cloud can only access your cluster when you provide the encryption key. DevSpace CLI securely stores your encryption key on your local machine and only sends it to DevSpace Cloud when running operations where DevSpace Cloud needs cluster access, e.g. when creating Spaces with `devspace create space [space-name]`.

If you forgot your encryption key, you can run this command:
```bash
devspace reset key [cluster-name]
```
DevSpace CLI will ask you to provide a new key, encrypt a new cluster access token with it and send this new token to DevSpace Cloud.
