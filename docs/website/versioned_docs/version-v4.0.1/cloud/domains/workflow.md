---
title: Workflow
id: version-v4.0.1-workflow
original_id: workflow
---

In a [Space](../../cloud/spaces/what-are-spaces), applications can be accessed through [port-forwarding](../../development/port-forwarding) on localhost or with [ingresses](../../workflow-basics/deployment/domains) on a certain domain. On this page we cover how you can access a space through ingresses.  

## The concept of Domains

In DevSpace Cloud, spaces are by default not allowed to create an ingress for any domain (You can allow all domains by enabling the option "Allow all ingress hosts in namespace" in the [limits](../../cloud/clusters/limits)). Rather these domains have to be connected to the space. As soon as a domain is connected to a space, the user is allowed to create an ingress for this domain. This is to prevent domain collision in a cluster where several spaces would try to use the same domain name and a unique route could not be found. In addition, DevSpace Cloud can automtically create a SSL certificate for that domain through Let's Encrypt. Just check the 'Create Certificate' checkbox when connecting a new domain, make sure the domain points to the correct LoadBalancer IP and a certificate will be created automatically.  

If you have a connected cluster, DevSpace Cloud connects a sub domain of the configured Spaces Domain automatically to a new space. So for example, if you have the Spaces Domain `users.my-domain.com` under `Clusters -> Click on cluster -> Cluster Configuration -> Spaces Domain` configured, DevSpace Cloud will allow a newly created space `foo` to create an ingress for domain name `foo.users.my-domain.com`. If such a name is already used the domain name will have a random suffix like this `foo-asd34i6.users.my-domain.com`. This makes sure no other space uses the same domain. Cluster Admins are always able to connect any additional domain to any space (Navigate to `Clusters -> Click on cluster -> Spaces -> Click on space -> Domains -> Add Domain`).  

## Managing Domains &amp; Ingresses

DevSpace Cloud provides an easy to use UI to manage the domains and ingresses of a space. Navigate to `Clusters -> Click on cluster -> Spaces -> Click on space -> Domains` to see all currently connected domains and their ingresses.

> If you don't have any ingress for a space configured yet, you can also run `devspace open` in a project and the command will automatically create an ingress for you for the selected service

You can easily create, change and delete ingresses via this view for your space. DevSpace Cloud will change the ingresses in the background in the space. In order to create an ingress for a space please make sure you have at least one deployed service in the space.    
