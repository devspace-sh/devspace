---
title: devspace connect cluster
---

```bash
#######################################################
############ devspace connect cluster #################
#######################################################
Connects an existing cluster to DevSpace Cloud.

Examples:
devspace connect cluster
#######################################################

Usage:
  devspace connect cluster [flags]

Flags:
      --admission-controller   Deploy the admission controller (default true)
      --cert-manager           Deploy a cert manager (default true)
      --context string         The kube context to use
      --domain string          The domain to use
  -h, --help                   help for cluster
      --ingress-controller     Deploy an ingress controller (default true)
      --key string             The encryption key to use
      --name string            The cluster name to create
      --provider string        The cloud provider to use
      --use-domain             Use an automatic domain for the cluster (default true)
      --use-hostnetwork        Use the host netowkr for the ingress controller instead of a loadbalancer
```
