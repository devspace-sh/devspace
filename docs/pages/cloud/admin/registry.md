---
title: Private docker registry
---

DevSpace Cloud allows your team to easily use a managed private docker registry. You can check the status of the registry with all the images in the Admin -> Registry tab. If you haven't deployed the registry yet, you can also deploy it there

> Keep in mind that the users have to rerun `devspace login` to automatically login to the registry, if it wasn't deployed before

## Registry Configuration

DevSpace Cloud will by default deploy the registry under the same domain as the control plane runs, so if your control plane runs under the domain https://devspace.my-domain.com, you can also push images to the docker registry devspace.my-domain.com. To change the domain, you have to redeploy the docker registry (this will not erase any images) by deactivating the registry and redeploying it with a different domain name. If you use another domain name, make sure you set an A DNS record that points to the external ip of the load balancer deployed by DevSpace Cloud.  

## Registry Access

In DevSpace Cloud, users only have access to certain registry namespaces:
- Admins can access all namespaces and repositories
- Non Admins can access their username namespace (e.g. a user named 'devspaceuser' can push and pull images in devspace.my-domain.com/devspaceuser/*)
- Team members can access their team namespace (e.g. a team member in the team named 'myteam' can push and pull images in devspace.my-domain.com/myteam/*)

Deleting certain images and tags is possible via the DevSpace Cloud UI under the tab 'Images'. Admin users can also see all repositories in the Admin -> Registry tab.

## Registry Maintenance

By default DevSpace Cloud will trigger a garbage collection in the registry every 48 hours. It will also delete the complete namespace of users and teams that get deleted. However, in order to make sure the registry doesn't run out of space, you should regularily delete unneeded images and repositories to allow the garbage collection to free up space. Also choose the registry size wisely and depending on your team size and image size.  
