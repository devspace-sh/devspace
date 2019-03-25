---
title: Add a container
---

If you want to add a container to the chart, you have two options:
1. define a new component in the chart (see [add custom component](/docs/customization/add-component)). This will result in a new kubernetes [deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) or [statefulset](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) (if the component has a persistent volume mount defined)
2. add a container to an existing component in `chart/values.yaml`. This will essentially add the container to the components deployment.

On this page the second option will be shown.

> If you just want to add a kubernetes yaml to the chart take a look at [add custom kubernetes files](/docs/customization/custom-manifests)

## Add container to an existing component

After initializing your project, your `chart/values.yaml` should look similar to: 

```yaml
components:
- name: default
  containers:
  - image: dscr.io/myuser/devspace
    resources:
      limits:
        cpu: "300m"
        memory: "300Mi"
        ephemeralStorage: "1Gi"
    # Environment variables
    env: []
  service:
    name: external
    type: ClusterIP
    ports:
    - externalPort: 80
      containerPort: 3000

...
```

Adding a new container is fairly simple. In this case we add a sidecar container (see [connect to google cloud sql](https://cloud.google.com/sql/docs/mysql/connect-kubernetes-engine)) so we can access google cloud sql:

```yaml
components:
- name: default
  containers:
  - image: dscr.io/myuser/devspace
    ...
  - image: gcr.io/cloudsql-docker/gce-proxy:1.11
    # Optional container command
    command: ["/cloud_sql_proxy",
              "-instances=<INSTANCE_CONNECTION_NAME>=tcp:3306",
              "-credential_file=/secrets/cloudsql/credentials.json"]
    # Optional container mounts
    volumeMounts:
    - containerPath: /secrets/cloudsql
      volume:
        name: cloudsql-instance-credentials
        readOnly: true
  service:
    name: external
    type: ClusterIP
    ports:
    - externalPort: 80
      containerPort: 3000

volumes:
- name: cloudsql-instance-credentials
  secret:
    secretName: cloudsql-instance-credentials
...
```

Since containers share the same network space, you have to be careful that the containers do not listen on the same ports, because they could be blocked by the other container.
