---
title: Teams
---

A team in DevSpace Cloud consists of a team owner, team admins and team members. A team will have its own docker registry namespace, where all team members can push and pull images from. A team can also own clusters.

## Create a team

Navigate in the DevSpace Cloud UI to the team tab and press the 'Create Team' button. Choose a name for the team and press 'Create'. You have now created a team and can invite team members.

## Create a team cluster

After you have [connected a cluster](https://devspace.cloud/docs/cloud/clusters/connect) to DevSpace Cloud, go to the clusters tab and press the 'Change to team cluster' button. Select the team you want to transfer ownership to and press 'Change Ownership'. DevSpace Cloud will add all users in the cluster to the team, if they are not already in the team.

> After changing the ownership of a cluster to a team, it is NOT possible to change the ownership back to a user!

## Deleting a team

If you want to delete a team, go to the teams tab and press 'Delete'. Bear in mind that deleting a team will delete ALL team clusters, spaces in those clusters and every docker image in the team registry namespace.
