# DevSpace Examples

This folder contains several examples how to use the devspace cli for various use cases. Below is a table with the examples:  

| Name | Builder | Registry | Deployment | Description |
|:------|:----------:|:----------:|:----------:|:-------------|
| [`quickstart`](https://github.com/covexo/devspace/tree/master/examples/quickstart) | docker | remote registry | helm | Simple nodejs example how to use devspace with helm |
| [`quickstart-kubectl`](https://github.com/covexo/devspace/tree/master/examples/quickstart-kubectl) | docker | remote registry | kubectl | Simple nodejs example how to use devspace with kubectl apply |
| [`minikube`](https://github.com/covexo/devspace/tree/master/examples/minikube) | minikube-docker | local registry | helm | Minikube example with local registry |
| [`offline-development`](https://github.com/covexo/devspace/tree/master/examples/offline-development) | minikube-docker | local registry | helm | Example how to develop without internet connection |
| [`php-mysql-example`](https://github.com/covexo/devspace/tree/master/examples/php-mysql-example) | docker | remote registry | helm | Example how to easily deploy php and mysql |
| [`microservices`](https://github.com/covexo/devspace/tree/master/examples/microservices) | docker | remote registry | helm & kubectl | Example with simple nodejs and php application that interact |
| [`kaniko`](https://github.com/covexo/devspace/tree/master/examples/kaniko) | kaniko | remote registry | helm | Example how to use kaniko instead of docker |
| [`kaniko-minikube`](https://github.com/covexo/devspace/tree/master/examples/kaniko-minikube) | kaniko | local registry | helm | Example how to use kaniko locally instead of docker |
