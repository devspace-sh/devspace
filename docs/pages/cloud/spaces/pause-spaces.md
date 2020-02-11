---
title: Pause Spaces
---

Pausing spaces is a feature of DevSpace Cloud where all ReplicaSets, Deployments and StatefulSets will be scaled to 0. Additionally, the allowed pod amount in the resource quota will be set to 0 and if configured, all existing pods in that namespace are killed. On space resume, DevSpace Cloud will rescale the resources to their previous replica amount. This is helpful to save infrastructure and resource cost if a space is currently not needed and should not be deleted completely.  

## Pausing a space

In the UI navigate to the Spaces view and click on the 'Pause' button. After a space is paused, you are able to resume the space via the UI or via DevSpace commands such as `devspace dev`, `devspace deploy`, `devspace logs` and `devspace enter` automatically.  

## Automatically pause a space

DevSpace can automatically pause spaces based on the last activity in that space. The last activity of a space is determined by calculating the last time a space was used with the DevSpace. Commands like `devspace dev`, `devspace deploy`, `devspace logs` and `devspace enter` automatically signal DevSpace Cloud that the space is still being used. These commands also automatically resume a space if it was paused previously.  

To configure if a space should be paused automatically and the timeout after which a space should be paused, navigate to the [Limits](../../cloud/spaces/resource-limits) view. There will be a section called **Sleep Mode** which allows you to configure these settings for individual spaces, users and clusters.  
