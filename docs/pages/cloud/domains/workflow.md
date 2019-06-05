---
title: Workflow
---

In a Space, applications can be accessed through [port-forwarding](/docs/development/port-forwarding) on localhost or with [ingresses](/docs/workflow-basics/deployment/domains) on a certain domain. On this page we cover how you can access a space through ingresses.  

## The concept of Domains

In DevSpace Cloud, spaces are by default not allowed to create an ingress for any domain (You can allow all domains by setting the option "Allow all ingress hosts in namespace" to true). Rather these domains have to be connected to the space. This is to prevent domain collision in a cluster where several spaces would try to use the same domain name and a unique route would not be found. So for each domain that is connected to a space you are allowed to create an ingress for it.  

If you have a connected cluster, DevSpace Cloud connects a sub domain of the configured Spaces Domain automatically to a new space. So for example you have the Spaces Domain `users.my-domain.com` under `Clusters -> Click on cluster -> Cluster Configuration -> Spaces Domain` configured, DevSpace Cloud will allow a newly created space `foo` to create an ingress for domain name `foo.users.my-domain.com`. If such a name is already used the domain name will have a random suffix like this `foo-asd34i6.users.my-domain.com`. This makes sure no other space uses the same domain.  

Cluster Admins are always able to connect any additional domain to any space (Navigate to `Clusters -> Click on cluster -> Spaces -> Click on space -> Domains -> Add Domain`). For Domains connected via this method a HTTPS certificate can also be created automatically.  

## Managing Domains & Ingresses

DevSpace Cloud provides an easy to use UI to manage the domains and ingresses of a space. Navigate to `Clusters -> Click on cluster -> Spaces -> Click on space -> Domains` to see all currently connected domains and their ingresses.

> If you don't have any ingress for a space configured yet, you can run `devspace open` in a project and that command will automatically create an ingress for you for the selected service

You can easily create, change and delete ingresses via this view for your space. DevSpace Cloud will change the ingresses in the background in the space.  
