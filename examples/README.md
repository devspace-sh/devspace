# DevSpace Examples

This folder contains several examples how to use the DevSpace CLI for various use cases. Below is a table with the examples:  

| Name | Builder | Registry | Deployment | Description |
|:------|:----------:|:----------:|:----------:|:-------------|
| [`quickstart`](https://github.com/devspace-cloud/devspace/tree/master/examples/quickstart) | docker | remote registry | helm | Simple nodejs example how to use devspace with helm |
| [`quickstart-kubectl`](https://github.com/devspace-cloud/devspace/tree/master/examples/quickstart-kubectl) | docker | remote registry | kubectl | Simple nodejs example how to use devspace with kubectl apply |
| [`minikube`](https://github.com/devspace-cloud/devspace/tree/master/examples/minikube) | minikube-docker | no registry | helm | Minikube example with local registry |
| [`php-mysql-example`](https://github.com/devspace-cloud/devspace/tree/master/examples/php-mysql-example) | docker | remote registry | helm | Example how to easily deploy php and mysql |
| [`microservices`](https://github.com/devspace-cloud/devspace/tree/master/examples/microservices) | docker | remote registry | helm & kubectl | Example with simple nodejs and php application that interact |
| [`kaniko`](https://github.com/devspace-cloud/devspace/tree/master/examples/kaniko) | kaniko | remote registry | helm | Example how to use kaniko instead of docker |
| [`redeploy-instead-of-hot-reload`](https://github.com/devspace-cloud/devspace/tree/master/examples/redeploy-instead-of-hot-reload) | docker | remote registry | kubectl | Example how to use devspace to redeploy on changes instead of hot reloading |
