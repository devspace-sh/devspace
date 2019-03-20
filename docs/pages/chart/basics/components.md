---
title: Components
---

On this page the DevSpace helm chart components (defined in `chart/values.yaml`) are explained.

> If you just want to quickly add a database like mysql, postgres, mogodb etc. you can checkout the [predefined components](/docs/customization/predefined-components)

> If you just want to add a kubernetes yaml to the chart take a look at [add custom kubernetes files](/docs/customization/custom-manifests)

## Component structure

In the DevSpace helm chart a component consists of several parts:

### containers:
The containers that will run together in a single kubernetes pod. Containers share volumes and the network. For more information about pods and how containers run inside pods take a look at [Pods](https://kubernetes.io/docs/concepts/workloads/pods/pod/)

You can specify several options for each container:
- image: the docker image to use for the container (if the docker image matches an image defined in `.devspace/config.yaml` the tag will be replaced during `devspace deploy/dev`)
- env: the environment variables used for this container (see [configure environment variables](/docs/customization/environment-variables) for more information)
- resources: the resources the container should run with
- volumeMounts: the paths within the container that should be mounted from a volume (see [persistent volumes](/docs/customization/persistent-volumes) for more information)

### service:
For each component a [Service](https://kubernetes.io/docs/concepts/services-networking/service/) is created. A service is specified by a name and an array of externalPorts and containerPorts. Other components (or kubernetes pods) can access the component then via the service name:

Take a look at this `values.yaml`:
```yaml
components:
# First component your app
- name: myapp
  containers:
  ...
# Second component the mysql database
- name: mysql
  containers:
  ...
  service:
    name: my-mysql-service
    ports:
    # Port on the service
    - externalPort: 1234
    # Port on the container
      containerPort: 3306
...
```

In this example the containers within the component myapp can access the mysql component via the address: `mysql://my-mysql-service:1234`
