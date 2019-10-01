---
title: Domains & Ingresses
id: version-v3.5.18-domains
original_id: domains
---

In Kubernetes, traffic from outside the cluster is being routed like this:
```bash
  Internet-Traffic   ||    Cluster-internal traffic
Domain / Public IP --||--> Ingress --> Service --> Pod/Container
```
If you want to make a deployment available on a domain, you need to:
1. [Configure a service](#configure-services) for this deployment
2. [Create an ingress](#configure-ingresses) that maps the traffic from your domain to the service you created in step 1

## Configure services
Services in Kubernetes provide a stable IP address (and/or cluster-internal DNS name) for your deployments although the deployed pods and containers might be deleted and re-created (e.g. when you update or scale deployments).

How to configure a service for one of your deployments, depends on the kind of deployment:
- `component` - [configure the service in your devspace.yaml](../../deployment/components/configuration/service)
- `kubectl` - [create a service manifest](#create-a-service-manifest) and add it to your `manifests`
- `helm` 
  - **local chart**: [create a service manifest](#create-a-service-manifest) and add it to the `templates/` folder of the chart
  - **chart from a registry**: set the appropriate config option in your `values.yaml` (depends on the Helm chart you are using)

## Configure ingresses
Ingresses define how external traffic from outside the Kubernetes cluster will be routed once it arrives at the cluster.

There are two options to configure an ingress:
1. Configure an ingress [using the UI of DevSpace Cloud](#configure-ingresses-with-devspace-cloud-ui)
2. Configure an ingress [manually](#configure-ingresses-manually)

### Configure ingresses with DevSpace Cloud UI
This obviously only works if you use DevSpace CLI in combination with DevSpace Cloud. To configure an ingress with DevSpace Cloud, you need to connect a domain to the respective Space:
1. Go to: [https://app.devspace.cloud/spaces](https://app.devspace.cloud/spaces)
2. Open the tab "Domains" for the Space you want to connect the domain to
3. Connect a domain as explained in the UI (if there is not one already)
4. Run `devspace open` in your project and select the service you want to connect to.

### Configure ingresses manually
If you are not using DevSpace Cloud, you will need to manually create ingresses. The following yaml show how the manifest for an ingress can look like:
```yaml
# my-ingress.yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: devspace-ingress
  ...
spec:
  rules:
  - host: my-domain.tld
    http:
      paths:
      - backend:
          serviceName: my-service
          servicePort: 8080
  tls:
  - hosts:
    - my-domain.tld
  ...
```
This ingress defines the rule that traffic on `http://my-domain.tld:80/` should be routed to the service `my-service` on port `8080`. Additionally, the hostname `my-domain.tld` is added within the `tls` section to define that `HTTPS` traffic will also be routed according to the rule defined above.

> To use TLS and route HTTPS traffic, you will need an appropriately configure ingress controller as well as an SSL certificate which can be automatically created by the [Kubernetes cert-manager](https://github.com/jetstack/cert-manager).
