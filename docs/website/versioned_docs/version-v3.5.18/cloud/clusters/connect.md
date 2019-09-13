---
title: Connect external clusters
id: version-v3.5.18-connect
original_id: connect
---

DevSpace Cloud provides the option to connect existing Kubernetes clusters for free. This will enable yout to create [spaces](/docs/cloud/spaces/what-are-spaces) in any kubernetes cluster. After connecting a cluster, you will be able to:
- create [spaces](/docs/cloud/spaces/what-are-spaces) in that cluster
- add other [users](/docs/cloud/clusters/users) to the cluster (by sending an invite link)
- set [Space limits](/docs/cloud/spaces/resource-limits) for the cluster, users and spaces

## Connect a new cluster
You can connect any Kubernetes cluster, by running the following command:
```bash
devspace connect cluster
```

DevSpace CLI will ask you a couple of questions and connect your cluster. The output will look similar to this one:
```bash
$ devspace connect cluster    
? Please enter a cluster name (e.g. my-cluster) my-cluster
? Which kube context do you want to use  [Use arrows to move, type to filter]
  kubectl-context-1
  kubectl-context-2
> current-kubect-context
  kubectl-context-3
? Please enter a secure encryption key for your cluster credentials ******** # Choose a password-like key for encrypting your cluster credentials
? Please re-enter the key ********                    
[done] √ Initialized cluster
? Should the ingress controller use a LoadBalancer or the host network?  [Use arrows to move, type to filter]
> LoadBalancer (GKE, AKS, EKS etc.)                   
  Use host network
[done] √ Deployed ingress controller                  
[done] √ Deployed admission controller                
[done] √ Deployed cert manager                        
[done] √ The domain '*.my-cluster.my.devspace.host' has been successfully configured for your clusters spa
ces and now points to your clusters ingress controller. The dns change however can take several minutes to
 take affect
[done] √ Successfully connected cluster to DevSpace Cloud.
```

## List connected clusters
To get a list of all connected clusters, run this commend:
```yaml
devspace list clusters
```

## Remove a connected cluster
To remove a connected cluster from DevSpace Cloud, simply run:
```bash
devspace remove cluster [cluster-name]
```

The command will ask you if you want to delete the cluster services (Ingress Controller, Cert Manager and Admission Controller) and all spaces. If you choose to delete the cluster services and spaces, all kubernetes resources created by DevSpace Cloud will be **permanently** removed from the cluster.
