---
title: Add custom components
id: version-v3.5.18-add-custom-components
original_id: add-custom-components
---

If you have understood the basic structure of a [component](../../deployment/components/what-are-components), adding another component is quite easy. After initializing your project, your `devspace.yaml` should look like this: 

```yaml
deployments:
- name: default
  component:
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
      - port: 80
...
```

Let's say you want to add a mysql component manually instead of using one of the [predefined components](../../deployment/components/add-predefined-components). You could simply add a new deployment with a similar definition as the default component:

```yaml
deployments:
- name: default
  ...
- name: mysql
  component:
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
      - port: 3306
```

You can redeploy the application now with `devspace deploy` and you should be able to access the mysql database within your default component via: `mysql://root:yourpassword@mysql-service-name:3306/mydatabase`.  

However the mysql database is currently running without persistent the data. Add a persistent volume path to the component and run `devspace deploy` again with this configuration:

```yaml
deployments:
- name: default
  ...
- name: mysql
  component:
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
          subPath: /mysql
    service:
      name: mysql-service-name
      ports:
      - port: 3306
    volumes:
    - name: mysql-data
      size: "5Gi"
```
