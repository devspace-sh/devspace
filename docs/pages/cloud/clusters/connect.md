---
title: Connect external clusters
---

DevSpace Cloud provides the option to connect existing Kubernetes clusters. After connecting a cluster, you will be able to:
- add users to the cluster (by sending an invite link)
- set user and Space limits for the cluster

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
? DevSpace will automatically create an ingress for each space, which base domain do you want to use for the created space? DevSpace will automatically create an ingress for each space, which base domain do you want to use for the created spaces? (e.g. users.test.com) dev.my-domain.tld
[done] √ Please create an A dns record for '*.dev.my-domain.tld' that points to external-ip of loadbalancer service 'devspace-cloud/nginx-ingress-controller'. Run `kubectl get svc nginx-ingress-controller -n devspace-cloud` to view the service
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
