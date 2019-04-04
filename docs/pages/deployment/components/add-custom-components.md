---
title: Add custom components
---

Run the following command to add a custom component to your deployments:
```bash
devspace add deployment [deployment-name] --component=[component-name]
```
After adding a component, you need to manually redeploy in order to start the newly added component together with the remainder of your deployments.
```bash
devspace deploy
```

On this page a custom [component](/docs/chart/basics/components) is added to the DevSpace chart.

> If you just want to add a kubernetes yaml to the chart take a look at [add custom kubernetes files](/docs/customization/custom-manifests)

## Add a Component

If you have understood the basic structure of a [component](/docs/chart/basics/components), adding another [component](/docs/chart/basics/components) to your chart is easy. After initializing your project, your `chart/values.yaml` should look like: 

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

Let's say you want to add a mysql component manually (There is an even easier way using [predefined components](/docs/customization/predefined-components)). We will use the official mysql image from docker hub. Now add a new component:

```yaml
components:
- name: default
  ...
- name: mysql
  containers:
  - image: mysql:5.7
    env:
    - name: MYSQL_ROOT_PASSWORD
      value: "yourpassword"
    - name: MYSQL_DATABASE
      value: "mydatabase"
  service:
    name: mysql-service-name
    ports:
    - externalPort: 3306
      containerPort: 3306
```

You can redeploy the application now with `devspace deploy` and you should be able to access the mysql database within your default component via: `mysql://root:yourpassword@mysql-service-name:3306/mydatabase`.  

However the mysql database is currently running without persistent the data. Add a persistent volume path to the component and run `devspace deploy`:

```yaml
components:
- name: default
  ...
- name: mysql
  containers:
  - image: mysql:5.7
    env:
    - name: MYSQL_ROOT_PASSWORD
      value: "yourpassword"
    - name: MYSQL_DATABASE
      value: "mydatabase"
    volumeMounts:
    - containerPath: /var/lib/mysql
      volume:
        name: mysql-data
        path: /mysql
  service:
    name: mysql-service-name
    ports:
    - externalPort: 3306
      containerPort: 3306

volumes:
- name: mysql-data
  size: "5Gi"
```
