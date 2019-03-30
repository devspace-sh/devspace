---
title: Service
---

In short, there is three things you need to know:
1. Containers in the same pod can communicate via `localhost`.
2. If you want containers to communicate across components, you need to [define services for these components](#define-services-for-your-components).
3. If you want external traffic (incoming internet traffic) to connect to your services, you need to [configure an ingress for this service](#configure-the-ingress-for-a-domain).

[Learn more about Kubernetes networking.](#understand-kubernetes-networking)

## Define services for your components
You can define a service for each of your components by configuring `components[*].service` within your `chart/values.yaml`.
```yaml
components:
- name: default
  containers:
  - image: "dscr.io/username/nodejs-app"
  - image: "dscr.io/username/mysql"
  service:
    name: my-service
    ports:
    - externalPort: 80
      containerPort: 3000
    - externalPort: 3306
      containerPort: 3306
```
The example above would define a service called `my-service` for the component `default` and forward all traffic from `my-service:80` (cluster-internal DNS address) to the component `default` on port `3000` and all traffic from `my-service:3306` to the component `default` in port `3306`. 

> Service names **must** be unique across all components. The name `external` is (by default) used to connect domains. [Learn more about configuring an `external` service for a domain](#configure-an-external-service-for-a-domain)

Let's assume that, in the above config example, the `nodejs-app` container would run a web server on port `3000` and the `mysql` container would run a mysql server on port `3306`. That means that:
- you could connect to the nodejs web server via `my-service:80`
- you could connect to the mysql server via `my-service:3306`

> The name of a service acts as something like an internally used domain name which your containers can connect to.

<details>
<summary>
### View the specification for services
</summary>
```yaml
name: [a-z0-9-]{1,253}      # Name of the service (used for cluster-internal DNS)
type: ClusterIP             # Type of the service (only ClusterIP is supported)
ports:
- externalPort: [number]    # External port exposed by the service
  containerPort: [number]   # Port of the container that the service redirects traffic to
```
</details>

## Configure an `external` service for a domain
To make one of your services public on the internet, you need to [connect a domain to your Space](/docs/cloud/spaces/domains) which automatically creates an ingress for this domain within your Space. An ingress routes the traffic from your domain name to a service defined for one of your components. When connecting a domain, this ingress will by default route all traffic to the service `external` on port `80`. 

So if you want to add the domain `example.tld`, you need to:
1. [Connect the domain `example.tld` your Space](/docs/cloud/spaces/domains)
2. Configure a service with name `external` in one of your components.
    ```yaml
    components:
    - name: default
      containers:
      - image: "dscr.io/username/nodejs-app"
      service:
        name: external
        ports:
        - externalPort: 80
          containerPort: 3000
    ```
This would allow you to access `example.tld:80` and Kubernetes will connect you to the server that is running inside the `nodejs-app` container on port `3000`.

## Configuring ingresses for domains
Generally, [adding a domain to your Space](/docs/cloud/spaces/domains) will create an ingress for this domain automatically. You can view a list of ingresses in your Space with the following commands:
```bash
devspace use space [SPACE_NAME]
kubectl get ingress
```
Although ingresses for connected domains are created automatically, you modify them manually, e.g. with the following command:
```
kubectl edit ingress [INGRESS_NAME]
```

To learn more about networking in devspace cloud, take a look at [configure space networking](/docs/cloud/spaces/configure-networking).

> It is **NOT** recommended to add an ingress definition to your `chart/template/` folder because it makes it harder to share the Helm chart configuration with other developers.

---
## FAQ

<details>
<summary>
### How do I create high-availability services?
</summary>
If you want fault-tolerance for your services, you can [define that your components run in a replicated way](/docs/chart/customization/scaling). Generally, incoming traffic for a service will be forwarded to a randomly selected replica of the service's component. However, if one of the components become unhealthy, Kubernetes will automatically forward traffic to the other available replicas. To allow Kubernetes to know which of your containers are unhealthy, you need to [define health checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) 
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
