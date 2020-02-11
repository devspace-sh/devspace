---
title: Manage cluster users
---

If you have your own cluster connected to DevSpace Cloud, you are able to invite users to that cluster, that can create spaces. In order to invite an user to the cluster, navigate to the `Clusters -> Click on Cluster -> Invites` view and create a new invite link. Send the invite link to the user you want to invite and when he accepts the invitation he is added to your cluster.  

## Cluster Roles

There are 3 roles in a cluster that have different rights within that cluster:
1. **Owner** is the cluster owner and has the same permissions as an cluster admin. There has to be exactly one cluster owner, that can not be removed from the cluster.
2. **Admin** is allowed to add, change and delete users, change cluster configuration and manage cluster spaces.
3. **User** is allowed to mangage his own cluster spaces (up to his limit). He is not allowed to change any users or cluster configurations.

## Manage users

To manage the users of your connected clusters:
1. Open the Clusters view in DevSpace Cloud: [https://app.devspace.cloud/clusters](https://app.devspace.cloud/clusters)
2. Click on "Users" for the respective cluster


## Reset Cluster Key

There are two methods how you can reset your cluster key, if you or another cluster user has forgotten his cluster key:
1. Reinvite the user through the UI. (Please be aware that you cannot reinvite yourself to a cluster)
2. If you are a cluster admin, you can also run `devspace reset key [cluster-name]` and select the admin kube context of the cluster where you want to reset the key for.
