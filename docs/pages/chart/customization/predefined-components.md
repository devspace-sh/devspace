---
title: Add a predefined component
---

DevSpace provides some easy and ready to use predefined [components](/docs/chart/basics/components), such as mysql, postgres, mongodb and others. You can list all available components with:
```json
$ devspace list available-components

 Name       Description                                                                               
 mariadb    MariaDB is a community-developed fork of MySQL intended to remain free under the GNU GPL  
 mongodb    MongoDB document databases provide high availability and easy scalability                 
 mysql      MySQL is a widely used, open-source relational database management system (RDBMS)         
 postgres   The PostgreSQL object-relational database system provides reliability and data integrity  
 redis      Redis is an open source key-value store that functions as a data structure server
```

> If you want to add an custom container image to the chart take a look at [add custom component](/docs/customization/add-component)

> If you just want to add a kubernetes yaml to the chart take a look at [add custom kubernetes files](/docs/customization/custom-manifests)

## Add a predefined component

Make sure you are at the root of your devspace project and have initialized the project with `devspace init`. Then run the following command in your project:
```bash
devspace add component mysql
```

You will be asked several questions about the component you want to add. Afterwards take a look at your `chart/values.yaml`:
```yaml
components:
- containers:
  - env:
    - name: MYSQL_ROOT_PASSWORD
      value: mypassword
    - name: MYSQL_DATABASE
      value: mydatabase
    image: mysql:5.7
    resources:
      limits:
        cpu: 100m
        ephemeralStorage: 1Gi
        memory: 200Mi
    volumeMounts:
    - containerPath: /var/lib/mysql
      volume:
        name: mysql-data
        path: /mysql
  name: mysql
  service:
    name: mysql
    ports:
    - containerPort: 3306
      externalPort: 3306
- name: default
  ...

# Define persistent volumes here
# Then mount them in containers above
volumes:
- name: mysql-data
  size: 5Gi

...
```

As you can see devspace has added the component. Now redeploy your application with `devspace deploy` and you should be able to access the mysql database **within** your default component via: `mysql://root:yourpassword@mysql:3306/mydatabase`.  
