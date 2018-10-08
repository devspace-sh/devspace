---
title: 2. Starting your DevSpace
---

To start your DevSpace, open the terminal (inside your IDE or in a separate window) and run the following command directly inside your existing project:
```bash
devspace up
```

This command will: 
1. ask some basic configuration questions,
2. create a Dockerfile, a helm chart and the devspace config (see below),
3. start a Tiller server (if necessary) and a private Docker registry (if wanted, you can also use any other registry) in your Kubernetes cluster,
4. build your Dockerfile and deploy the helm chart in chart/,
5. start port-forwarding and real-time code synchronization,
6. and open a terminal session.

## File Structure
The `devspace up` command internally calls `devspace init` which creates the following files for you:
```bash
YOUR_PROJECT_PATH/
|
|-- Dockerfile
|
|-- chart/
|   |-- Chart.yaml
|   |-- values.yaml
|   |-- templates/
|       |-- deployment.yaml
|       |-- service.yaml
|       |-- ingress.yaml
|
|-- .devspace/
|   |-- .gitignore
|   |-- cluster.yaml
|   |-- config.yaml
```  

**Note:** Don't worry, you can simply run `devspace reset` to reset your project to its original state (see [Cleanup](/docs/getting-started/cleanup.html)).
