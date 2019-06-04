---
title: Resource limits
---

# TODO @Fabian

## Resource Limits

Free spaces are limited in resources. In the free tier a space can use up to:
- 600m CPU
- 1Gi Memory
- 5Gi ephemeral storage
- 10Gi of persistent storage
- 6 pods (max 3 container per pod)
- 30 config maps
- 30 secrets
- 4 ingresses

The default values if not other specified are:
- 100m CPU per container
- 200Mi memory per container
- 1Gi ephemeral storage per container

## Space access restrictions

You can create most of the kuberentes resources in a space including:
- configmaps
- serviceaccounts
- roles
- rolebindings
- pods
- replicasets
- deployments
- replicationcontrollers
- statefulsets
- services
- secrets
- endpoints
- horizontalpodautoscalers
- cronjobs
- jobs
- persistentvolumeclaims
- ingresses

However certain actions are prohibited to ensure namespace isolation:
- node port services
- loadbalancer services
- cluster level resources
- elevated containers and host paths
- networkpolicies, limitranges and resourcequotas
