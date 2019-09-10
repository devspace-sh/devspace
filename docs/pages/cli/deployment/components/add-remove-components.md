---
title: Predefined & Custom Components
---

Run the following command to add a predefined component to your deployments:
```bash
devspace add deployment [deployment-name] --component=[component-name]
```
After adding a component, you need to manually redeploy in order to start the newly added component together with the remainder of your deployments.
```bash
devspace deploy
```

## List of predefined components
DevSpace provides the following predefined components:
- mariadb
- mongodb
- mysql
- postgres
- redis

## Example: Adding mysql
To add mysql as predefined component to your deployments, run this command:
```bash
devspace add deployment database --component=mysql
```

DevSpace will ask a couple of questions before adding the component.
```bash
? Please specify the mysql version you want to use 5.7
? Please specify the mysql root password my-password-123
? Please specify the mysql database to create on image startup my_database
? Please specify the database size in Gi 5Gi
[done] √ Successfully added database as new deployment
```

After adding the mysql component as a deployment, your `devspace.yaml` will contain a section similar to this one:
```yaml
deployments:
- name: database
  component:
    containers:
    - image: mysql:5.7
      env:
      - name: MYSQL_ROOT_PASSWORD
        value: my-password-123
      - name: MYSQL_DATABASE
        value: my_database
      volumeMounts:
      - containerPath: /var/lib/mysql
        volume:
          name: mysql-data
          subPath: /mysql
    volumes:
    - name: mysql-data
      size: 5Gi
    service:
      name: mysql
      ports:
      - port: 3306
- ... # your previously defined deployments
```

> DevSpace always **prepends** new components in the deployments array within your `devspace.yaml`, so that the components you need will be deployed before your application will be started.

After adding the mysql database component, you need to redeploy:
```bash
devspace deploy
```

Now, you will be able to access your mysql database from other containers within your Kubernetes namespace using the following connection variables:

| Connection Variable | Value |
| ---:|---|
| Host | `mysql` |
| Port | `3306` |
| Username | `root` |
| Password | `my-password-123` |
| Database | `my_database` |
| Connection String | `mysql://root:my-password-123@mysql:3306/my_database` |


Add custom components
---

If you have understood the basic structure of a [component](/docs/deployment/components/what-are-components), adding another component is quite easy. After initializing your project, your `devspace.yaml` should look like this: 

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

Let's say you want to add a mysql component manually instead of using one of the [predefined components](/docs/deployment/components/add-predefined-components). You could simply add a new deployment with a similar definition as the default component:

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
