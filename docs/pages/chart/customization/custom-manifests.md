---
title: Add custom Kubernetes files
---

This page shows how you can add custom kubernetes yamls to the chart. 

## Add a custom manifest to the chart

You can just copy kubernetes yamls into the `chart/templates/custom` folder and they will be deployed along side your application on `devspace deploy`. For example copy this kuberentes deployment to a file in `chart/templates/custom/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
```

Now run `devspace deploy` and `kubectl get pods`, you should see that the specified `deployment.yaml` was deployed:

```bash
$ kubectl get po
NAME                                READY   STATUS    RESTARTS   AGE
default-6cd95d79cc-qqr6q            1/1     Running   0          9s
nginx-deployment-78f5d695bd-vhrnp   1/1     Running   0          17s
tiller-deploy-568f8684c5-mt6g4      1/1     Running   0          2m
```

> It is advised to use the `chart/templates/custom` folder instead of the `chart/templates` folder, because on `devspace update chart` the custom manifests could be deleted in `chart/templates`
