---
title: Service
---

In short, there is three things you need to know:
1. Containers in the same component will be started within a [Kubernetes pod](#what-are-pods) which allows the containers of same component to communicate via `localhost`.
2. If you want containers to communicate across components, you need to [define services for these components](#define-services-for-your-components).
3. If you want to connect a domain to a service, you need to [configure an ingress for this service](/docs/workflow-basics/deployment/domains#configure-ingresses).

## Define services for your components
You can define a service for a component by configuring the `service` section of this component within your `devspace.yaml`.
```yaml
deployments:
- name: backend-api
  component:
    containers:
    - image: "dscr.io/username/nodejs-app"
    service:
      ports:
      - port: 80
        containerPort: 3000
- name: database
  component:
    containers:
    - image: "dscr.io/username/mysql"
    service:
      ports:
      - port: 3306
```
The example above would define two services:
1. Service `backend-api` which forwards all traffic from `backend-api:80` (cluster-internal DNS address) to the component `backend-api` on port `3000`
2. Service `database` which forwards all traffic from `database:3306` (cluster-internal DNS address) to the component `database` on port `3306`

With the above services, it would be possible that our containers within the `backend-api` component could connect to the MySQL server running in `database` with this connection string: `mysql://USERNAME:PASSWORD@database:3306/DB_NAME`

> Service names **must** be unique across all components. If you do not specify a name for the service, it will have the same name as the component. Service names can be seen as cluster-internal domains that allow containers to access containers from other components.

<details>
<summary>
### View the specification for services
</summary>
```yaml
name: [a-z0-9-]{1,253}      # Name of the service (used for cluster-internal DNS, default: component name)
type: ClusterIP             # Type of the service (default: ClusterIP, only ClusterIP is supported)
ports:
- port: [number]            # External port exposed by the service
  containerPort: [number]   # Port of the container that the service redirects traffic to (default: value of port option)
externalIPs:
- 123.45.67.890             # ExternalIP to expose the service on (discouraged)
```
</details>


---
## FAQ

<details>
<summary>
### How do I create high-availability services?
</summary>
If you want fault-tolerance for your services, you can [define that your components run in a replicated way](/docs/cli/deployment/components/configuration/scaling). Generally, incoming traffic for a service will be forwarded to a randomly selected replica of the service's component. However, if one of the components become unhealthy, Kubernetes will automatically forward traffic to the other available replicas. To allow Kubernetes to know which of your containers are unhealthy, you need to [define health checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) 
</details>

<details>
<summary>
### How should containers within the same component communicate?
</summary>
DevSpace automatically defines a pod for each of your components, i.e. all containers that you define in the same components in your `chart/values.yaml` will be in the same pod and can communicate via `localhost`.
</details>

<details>
<summary>
### How do should containers communicate across different components?
</summary>
If you want a container A to access a container B running inside another component, you should [define a service](#define-services-for-your-components) pointing to container B.
</details>

<details>
<summary>
### What are pods?
</summary>
Pods are groups of containers which share the same network stack. That means that containers within the same pod can communicate via `localhost`. It also means that two containers cannot use the same port for an application, i.e. if one containers starts an application on port 3000, all other containers within the same pod cannot use this port anymore.

Each pod within your Space will get a cluster-internal IP address of the format `10.X.X.X`.
</details>

<details>
<summary>
### What are services?
</summary>
Services are used for inter-pod communication. Each service within your Space will get a cluster-internal IP address of the format `10.X.X.X` which can be used to connect to the service. However, you should not connect directly to this IP address. Instead, you should connect to the DNS name of this service which is simply the name of the service.

> Altough you can directly use the IP addresses of your containers/pods or of your services for internal communication, you should use the (DNS) name of a service instead because the IP addresses might change.
</details>
