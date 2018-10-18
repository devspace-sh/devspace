# Php Mysql example

This example shows you how to develop a small php web application that uses mysql as a database.

# Step 0: Prerequisites

In order to use this example, make sure you have docker installed and a docker registry where you can push to (hub.docker.com, gcr.io etc.). Make sure you are logged in to the registry via `docker login`.  

Exchange the image name in `.devspace/config.yaml` under `images.default.name` with the image name you want to use. Do **not** add a tag to this image name, because this will be done at runtime automatically.  

## Optional: Use self hosted cluster (minikube, GKS etc.) instead of devspace-cloud

By default, this example will deploy to the devspace-cloud, a free managed kubernetes cluster. If you want to use your own cluster instead of the devspace-cloud as deployment target, make sure `kubectl` is configured correctly to access resources on the cluster. Then just erase the `cluster` section in the `.devspace/config.yaml` and devspace will use your current `kubectl` context as deployment target.  

# Step 1: Start the devspace

To deploy the application to the devspace-cloud simply run `devspace up`. The output of the command should look similar to this: 

```
INFO]   Building image 'fabian1991/devspace' with engine 'docker'
[DONE] √ Authentication successful (hub.docker.com)
Sending build context to Docker daemon  7.168kB
Step 1/6 : FROM php:7.1-apache-stretch
 ---> 93e6fb4b13e1
Step 2/6 : ENV PORT 80
 ---> Using cache
 ---> 788a58416826
Step 3/6 : EXPOSE 80
 ---> Using cache
 ---> b01729d4ade9
Step 4/6 : RUN docker-php-ext-install mysqli && docker-php-ext-enable mysqli
 ---> Running in 1dbd0ba85b71
Configuring for:
PHP Api Version:         20160303
Zend Module Api No:      20160303
Zend Extension Api No:   320160303
[...]
warning: mysqli (mysqli.so) is already loaded!

 ---> d4942a93c915
Step 5/6 : COPY . /var/www/html
 ---> dbb818ed4318
Step 6/6 : RUN usermod -u 1000 www-data;     a2enmod rewrite;     chown -R www-data:www-data /var/www/html
 ---> Running in 1c90fe728588
Enabling module rewrite.
To activate the new configuration, you need to run:
  service apache2 restart
 ---> 99eb800d832d
Successfully built 99eb800d832d
Successfully tagged fabian1991/devspace:HKgauKH
The push refers to repository [docker.io/fabian1991/devspace]
aff6c1858e21: Pushed
d1bd441544af: Pushed
60f45585ecc1: Pushed
bc0dfe6b56ad: Layer already exists
f82ba3fb9cea: Layer already exists
c6a1866bd1a0: Layer already exists
c9b57a1cfeb1: Layer already exists
1e97bcb161b3: Layer already exists
369e6fd590f3: Layer already exists
1805144065e1: Layer already exists
b6311cdc5fb6: Layer already exists
e30181a94bbf: Layer already exists
481da43a1302: Layer already exists
a4ace4ed0385: Layer already exists
fd29e0f8792a: Layer already exists
687dad24bb36: Layer already exists
237472299760: Layer already exists
HKgauKH: digest: sha256:6530af9474b2e9b8b1bfc6986288b4dcd34fca5365ffee60a2f6f63de4327b80 size: 3868
[INFO]   Image pushed to registry (hub.docker.com)
[DONE] √ Done building and pushing image 'fabian1991/devspace'
[INFO]   Deploying devspace-default with helm
[DONE] √ Deployed helm chart (Release revision: 2)
[DONE] √ Successfully deployed devspace-default
[DONE] √ Port forwarding started on 3000:80
[DONE] √ Sync started on /go-workspace/src/github.com/covexo/devspace/examples/php-mysql-example <-> /var/www/html (Pod: test/devspace-default-55c89799d5-x8rvl)
root@devspace-default-55c89799d5-x8rvl:/var/www/html#
```

The command built your Dockerfile and pushed it to the target docker registry. Afterwards, it created a new kubernetes namespace for you in the devspace-cloud and deployed the `kube/deployment.yaml` to that namespace. It also created a new kubectl context for you. If you want to access kubernetes resources via kubectl in the devspace-cloud you can simply change your kubectl context via `devspace up --switch-context`. Now you can check the running pods via `kubectl get po`.

Furthermore a bi-directional sync was started between the local folder `/go-workspace/src/github.com/covexo/devspace/examples/php-mysql-example` and `/var/www/html` within the docker container. Whenever you change a file in either of those two folders the change will be synchronized. In addition the container port 80 was forwarded to your local port 3000.  

# Step 2: Start developing

Navigate in your browser to `localhost:3000` and you should a signin page. If you submit the form, the application will insert a new entry in the `Users` table of the mysql database. Try changing the `index.php` locally and reload the webpage and you should be able to see the changes immediately.
