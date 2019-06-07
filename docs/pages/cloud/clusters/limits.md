---
title: Configure Space Limits
---

When creating a [Space](/docs/cloud/spaces/what-are-spaces), DevSpace Cloud will create a namespace within a cluster and apply certain limits to it. In DevSpace Cloud there are Space Limits defined for each cluster, user and space:
- Cluster Space Limits are used as default Space Limits for new users (Can be changed in `Clusters -> Click On Cluser -> Limits`)
- User Space Limits are used as default Space Limits for new spaces created by the user (Can be changed in `Clusters -> Click On Cluser -> Users -> Change Limits`)
- Space Limits are actually applied to the namespace and applied during Space intialization (Can be changed in `Clusters -> Click On Cluser -> Spaces -> Click On Space -> Limits`)

Space Limits can only be changed by Cluster Admins and Owners. Cluster Users cannot change any of the above limits.

## How are limits applied during space creation?

There are two ways how a Space can be created:
1. By the cluster admin via `Clusters -> Click On Cluser -> Spaces -> Create Space`
2. By the user himself via `devspace create space` or the UI

When using the first way, the cluster admin is able to specify all the limits manually during creation. When the user himself creates a space, DevSpace Cloud will do the following:
1. Checks if the user is allowed to create a space for this cluster based on how many spaces he is allowed to create (Can be changed in `Clusters -> Click On Cluser -> Users -> Change Limits`)
2. Apply the users default space limits to that space (Can be changed in `Clusters -> Click On Cluser -> Users -> Change Limits`)
3. Create the space

## Defining Space Limits

There are several categories when defining Space Limits. DevSpace makes use of requests and limits when defining resource usage. For more information about the difference between those two, take a look at the [kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#resource-requests-and-limits-of-pod-and-container)

<details>
<summary>
### Limits per Namespace
</summary>
Limits in this section apply to the complete Space and are enforced through a [Resource Quota](https://kubernetes.io/docs/concepts/policy/resource-quotas/). 

**Max Limit CPU**: The max sum allowed of cores limits defined on all containers per namespace  
**Max Limit Memory**: The max sum allowed of memory limits defined on all containers per namespace  
**Max Limit Ephemeral Storage**: The max sum allowed of [ephemeral storage](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#local-ephemeral-storage) limits defined on all containers per namespace  
**Max Requests CPU**: The max sum allowed of cores requests defined on all containers per namespace  
**Max Requests Memory**: The max sum allowed of memory requests defined on all containers per namespace  
**Max Requests Ephemeral Storage**: The max sum allowed of [ephemeral storage](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#local-ephemeral-storage) requests defined on all containers per namespace  
**Max Requests Storage**: The max sum allowed of all requested storage trough persistent volume claims in a namespace  
**Max Pods**: The max number of pods allowed in a namespace  
**Max Services**: The max number of services allowed in a namespace  
**Max Persistent Volumes Claims**: The max number of persistent volume claims allowed in a namespace  
**Max Secrets**: The max number of secrets allowed in a namespace  
**Max Config Maps**: The max number of config maps allowed in a namespace  
**Max Ingresses**: The max number of ingresses allowed in a namespace  
**Max Roles**: The max number of roles allowed in a namespace  
**Max Roles Bindings**: The max number of roles bindings allowed in a namespace  
**Max Service Accounts**: The max number of service accounts allowed in a namespace  
**Custom resourcequota limits**: Custom resource quota limits that will be appended to the `spec.hard` section of the created resource quota (e.g. "count/customresource=10,count/customresource2=10")

</details>

<details>
<summary>
### Limits per Container
</summary>
Limits in this section apply to individually deployed containers and are enforced through a [Limit Range](https://kubernetes.io/docs/concepts/policy/limit-range/). 

**Default Limit CPU**: The default cpu limits to use if no limits is defined for the container  
**Default Limit Memory**: The default memory limits to use if no limits is defined for the container  
**Default Limit Ephemeral Storage** The default [ephemeral storage](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#local-ephemeral-storage) limits to use if no limits is defined for the container  
**Default Requests CPU**: The default cpu requests to use if no reuests is defined for the container  
**Default Requests Memory**: The default memory requests to use if no requests is defined for the container  
**Default Requests Ephemeral Storage** The default [ephemeral storage](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#local-ephemeral-storage) requests to use if no requests is defined for the container  
**Max Container CPU**: Maximum amount of cores allowed per container  
**Max Container Memory**: Maxmimum amount of memory allowed per container  
**Max Container Ephemeral Storage** Maxmimum amount of ephemeral storage allowed per container  
**Min Requests Container CPU**: The minimum requests allowed for container cpu  
**Min Requests Container Memory**: The minimum requests allowed for container memory  
**Min Requests Container Ephemeral Storage**: The minimum requests allowed for ephemeral Storage  
**Min Container CPU**: The minimum limits allowed for container cpu (Enforced by admission controller)  
**Min Container Memory**: The minimum limits allowed for container memory (Enforced by admission controller)  
**Min Container Ephemeral Storage**: The minimum limits allowed for ephemeral storage (Enforced by admission controller)  

</details>
<details>
<summary>
### Limits per Persistent Volume Claim
</summary>

Limits in this sections apply to each [persistent volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#lifecycle-of-a-volume-and-claim) and are enforced through a [Limit Range](https://kubernetes.io/docs/concepts/policy/limit-range/).  

**Max Limit Persistent Volume Claim**: The maximum limit of requested storage per persistent volume claim  
**Min Limit Persistent Volume Claim**: The minimum limit of requested storage per persistent volume claim  

</details>


<details>
<summary>
### Advanced Options
</summary>

**Enable Network Policy**: Deploys a network policy for the space that disallows pods in the namespace to communicate with other namespaces  
**Enable Admission Controller**: Marks the namespace for the admission controller to check for certain security problems within container specifications and enforces some limits  
**Enable Limit Range**: Deploys a limit range object into the space that enforces default limits and limits per container and persistent volume claim  
**Enable Resource Quota**: Deploys a resource quota object into the space that enforces namespace limits  
**Use Cluster Role for Service Account**: Uses the given cluster role to create a rolebinding for the default and user service account in the space  
**Use Ingress Class for ingresses**: Enforces the specified ingress class for all ingresses in the space  
**Allow all ingress hosts in namespace**: If true allows the user to specify any host in an ingress. If false only hosts that are added in the `Domains` are allowed as hosts for ingresses. Domains can only be added by cluster admins or are added by default if a cluster default space domain is configured.  
**Skip Admission controller pod security checks**: If false the admission controller skips potential security issue checks on deployed pods  
**Max Pod Container**: The maximum number of containers allowed per pod (Enforced by admission controller)  
**Max pod termination grace period in seconds**: The maximum allowed number of seconds to wait if a pod should be terminated  
**Pod Egress Bandwidth**: If specified automatically enforces the annotation "kubernetes.io/egress-bandwidth" on each deployed pod  
**Pod Ingress Bandwidth**: If specified automatically enforces the annotation "kubernetes.io/ingress-bandwidth" on each deployed pod  
**Empty Dir Storage Allowed**: If true allows pods to specify an empty dir volume (enforced by admission controller)  
**Empty Dir Storage Default Size**: The default size of the empty dir volume if none specified (enforced by admission controller)  
**Empty Dir Storage Max Size**: The maximum size allowed for empty dir volumes (enforced by admission controller)  
**Set Node Selector for pods**: Automatically makes sure the following node selector is set for each deployed pod (e.g. devspace.cloud/type=limited)  
**Set Tolerations for pods**: Automatically makes sure the following tolerations are set for each deployed pod (e.g. devspace.cloud/taint=limited)  

</details>

<details>
<summary>
### Templates
</summary>

In this section you can define any kubernetes resources that will be deployed at space creation. You can define pods, custom service accounts, role bindings, custom resources etc. here. The resources will be deployed via `kubectl apply -f` on space creation.  

</details>

## Changing Space Limits

You can change the Space Limits for clusters, users and spaces always at a later point aswell:

### Change Cluster Limits

Navigate to `Clusters -> Click on a Cluster -> Limits`. Change the limits and press apply. The following additional options exist:
- **Override all cluster users default limits** will override the default space limits of all users to the specified limits
- **Apply to all cluster spaces** will apply the limits to all cluster spaces and change (!) resource quotas, limit ranges, network policies and redeploy templates.
- **Force** This option is only necessary if apply the limit to spaces and you want to change the limits to a lower value than what is currently in use already by a space. For example a cluster space has 10 pods and you change the space limits to 5 pods, when checking this checkbox, devspace will apply those limits even though the space uses more than allowed resources currently. 

### Change User Limits

Navigate to `Clusters -> Click on a Cluster -> Users -> Change Limits`. Change the limits and press apply. The following additional options exist:
- **Apply to all existing user spaces** will apply the limits to all cluster spaces that were created by the user and change (!) resource quotas, limit ranges, network policies and redeploy templates if necessary.
- **Force** This option is only necessary if apply the limit to spaces and you want to change the limits to a lower value than what is currently in use already by a space. For example a cluster space has 10 pods and you change the space limits to 5 pods, when checking this checkbox, devspace will apply those limits even though the space uses more than allowed resources currently. 

### Change Space Limits

Navigate to `Clusters -> Click on a Cluster -> Spaces -> Click on a Space -> Limits`. Change the limits and press apply. This will change (!) resource quotas, limit ranges, network policies and redeploy templates if necessary. The following additional options exist:
- **Force** This option is only necessary if apply the limit to spaces and you want to change the limits to a lower value than what is currently in use already by a space. For example a cluster space has 10 pods and you change the space limits to 5 pods, when checking this checkbox, devspace will apply those limits even though the space uses more than allowed resources currently. 
