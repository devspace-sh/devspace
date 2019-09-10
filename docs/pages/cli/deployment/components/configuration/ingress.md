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
#TODO

### `labels`
#TODO

### `annotations`
#TODO

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
