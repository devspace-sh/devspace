---
title: Ingress (Domain)
---

To automatically create an ingress for a component, you can configure the `ingress` option for the component within the `devspace.yaml`.
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      rules:
      - host: my-static-host.tld
        tls: true
      - host: ${DYNAMIC_HOSTNAME}
        path: /login
```

> You need to define a `service` with at least one `port` to be able to use the `ingress` option.

[What are components?](/docs/cli/deployment/components/what-are-components)


## Ingress Rules

### `rules[*].host`
The `host` option expects a string stating the hostname (domain name) that the component should be made available on.

> DevSpace automatically makes sure that all hosts specified in `ingress.rules` are connected to the `service` of the component.

> If the `service` defined multiple `ports`, DevSpace will use the first one unless you specify the [`servicePort` option](#rules-serviceport).

#### Example: Defining Hosts
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      rules:
      - host: my-static-host.tld
      - host: ${DYNAMIC_HOSTNAME}
```


### `rules[*].tls`
The `tls` option expects either:
- a string stating the name of a Kubernetes secret which contains the TLS certificate to use for SSL
- a boolean to enable/disable TLS (an auto-generated name of a secret will be created referencing a Kubernetes secret containing the TLS certificate to use for SSL)

> This option takes precedence over the global option `ingress.tls` which sets the TLS option for all hosts.

#### Default Value For `tls`
```yaml
tls: false
```

#### Example: Enabling TLS for Single Hosts
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      rules:
      - host: my-static-host.tld
        tls: true
      - host: my-static-host2.tld
```


### `rules[*].path`
The `path` option expects a URL path which is used for routing. Only requests to this `path` will be forwarded to the service of this component.

#### Default Value For `path`
```yaml
path: /
```

#### Example: Enabling TLS
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      rules:
      - host: my-static-host.tld
        path: /login
```


### `rules[*].servicePort`
The `servicePort` option expects an integer stating the port of the service to which the traffic should be routed for the hostname stated in `host`.

> By default, DevSpace will automatically route to the first port specified under `service.ports` within this component.

#### Example: Custom Service Port
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
      - port: 8000
    ingress:
      rules:
      - host: my-static-host.tld
        servicePort: 8000
```


## Ingress Options

### `name`
The `name` option expects a string that will be used as a name for the ingress that is being created for this component.

> **The `name` field is optional.** By default, the component chart will name the ingress after the [component `service`](#TODO) it is referencing.

#### Example: Custom Name for Headless Service
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      name: custom-ingress-name
      tls: true
      rules:
      - host: my-static-host.tld
      - host: my-static-host2.tld
```
**Explanation:**  
Instead of the default name `frontend`, the ingress of this component would be named `custom-ingress-name`.

### `labels`
The `labels` option expects a map with Kubernetes labels. 

By default, the component chart sets a couple of labels following the best practices described in the Kubernetes documentation:
- `app.kubernetes.io/name: devspace-app`
- `app.kubernetes.io/component: [DEPLOYMENT_NAME]`

> You can specify additional labels using the `labels` option but the default / best practice labels will still be set for the component.

All additional labels will be added to the ingress created for this component.

#### Default Value For `labels`
```yaml
labels: []
```

#### Example: Additional Labels
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      tls: true
      rules:
      - host: my-static-host.tld
      - host: my-static-host2.tld
      labels:
        label1: label-value-1
        label1: label-value-2
```


### `annotations`
The `annotations` option expects a map with Kubernetes annotations. 

By default, the component chart sets a couple of annotations following the best practices described in the Kubernetes documentation:
- `helm.sh/chart: component-chart-vX.Y.Z`

> You can specify additional annotations using the `annotations` option but the default / best practice annotations will still be set for the component.

All additional annotations will be added to the ingress created for this component.

#### Default Value For `annotations`
```yaml
annotations: []
```

#### Example: Additional Annotations
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      tls: true
      rules:
      - host: my-static-host.tld
      - host: my-static-host2.tld
      annotations:
        annotation1: annotation-value-1
        annotation1: annotation-value-2
```


### `tls`
The `tls` option expects either:
- a string stating the name of a Kubernetes secret which contains the TLS certificate to use for SSL
- a boolean to enable/disable TLS (an auto-generated name of a secret will be created referencing a Kubernetes secret containing the TLS certificate to use for SSL)

#### Default Value For `tls`
```yaml
tls: false
```

#### Example: Enabling TLS for All Hosts
```yaml
deployments:
- name: frontend
  component:
    containers:
    - image: dscr.io/${DEVSPACE_USERNAME}/appfrontend
    service:
      ports:
      - port: 3000
    ingress:
      tls: true
      rules:
      - host: my-static-host.tld
      - host: my-static-host2.tld
```
