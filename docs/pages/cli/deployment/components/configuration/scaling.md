---
title: Scaling
---

There are two options to scale a component:
1. Manually scale the number of replicas
2. Configure horizontal auto-scaling

## Set number of replicas
To set the number of replicas, simply set the `replicas` parameter for a component within the `devspace.yaml`
```yaml
deployments:
- name: my-backend
  helm:
    componentChart: true
    values:
      replicas: 4
      containers:
      ...
```
Instead of just running the containers defined in `my-backend` with one pod, the above example would run `4` pods with the specified containers. Each of these pods run isolated, i.e. altough the containers of one pod can communicate via `localhost`, the containers of different pods cannot.

## Configure horizontal auto-scaling
To enable horizontal auto-scaling for a component, you just need to set `autoScaling.horizontal.maxReplicas` greater than the value `replicas` (see above). Additionally, you should configure one or multiple of the target value parameters, `averageCPU` and `averageMemory`. These target values define how the auto-scaler will set the number of replica to achieve an average CPU utilization and/or an average memory usage by the pods that will be scaled within this component.

```yaml
deployments:
- name: my-backend
  helm:
    componentChart: true
    values:
      replicas: 4
      autoScaling:
        horizontal:
          maxReplicas: 10
          averageCPU: 800m
    ...
```
The above example would create an [horizontal po auto-scaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) which is configured to:
- create at least 4 pods for the component
- scale the component up to a maximum of 10 pods
- observe the CPU usage of the pods and try to scale between 4 and 10 pods to achieve an average CPU utilization of 800m
