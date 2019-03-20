---
title: Configure Networking
---

In DevSpace cloud by default an [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) is created for each space that routes incoming traffic to an url specific to that space. By default this is https://your-space-name.devspace.host .  

You can look at the default ingress via [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/):
```bash
$ kubectl get ingress devspace-ingress 
NAME               HOSTS                       ADDRESS         PORTS     AGE
devspace-ingress   test-42bbdd.devspace.host   35.197.73.240   80, 443   23h 
```

# Understanding routing

In general incoming traffic is routed via this schema:
```
Internet -> DevSpace Cloud -> Ingress Controller -> Ingress -> Service -> Pod:Container          
```

Let's take a look at a standard `chart/values.yaml`:
```yaml
components:
- name: default
  containers:
  - image: dscr.io/youruser/devspace
  ...
  service:
    name: external
    ports:
    - externalPort: 80
      containerPort: 3000
...
```

This values.yaml tells devspace to create a service named `external` that listens on port 80 and redirects that port to port 3000 in the component. Now take a look at the default ingress:
```bash
$ kubectl get ingress devspace-ingress -o yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: devspace-ingress
  ...
spec:
  rules:
  - host: test-42bbdd.devspace.host
    http:
      paths:
      - backend:
          serviceName: external
          servicePort: 80
  tls:
  - hosts:
    - test-42bbdd.devspace.host
  ...
```

This definition tells the ingress controller to redirect traffic to a service called external on port 80. So this means traffic is routed to `dscr.io/youruser/devspace`like this:
```
Internet -> DevSpace Cloud -> Ingress Controller -> devspace-ingress -> external:80 -> dscr.io/youruser/devspace:3000    
```

# Change container port traffic is routed to

Changing the `service.ports.containerPort` will route traffic to your container on a different port (make sure your container is listening on 0.0.0.0:newport) and your `chart/values.yaml` looks like this:
```yaml
components:
- name: default
  ...
  service:
    name: external
    ports:
    - externalPort: 80
      containerPort: newport 
...
```

Then just run `devspace deploy` and the traffic will be routed like this:
```
Internet -> DevSpace Cloud -> Ingress Controller -> devspace-ingress -> external:80 -> dscr.io/youruser/devspace:newport    
```

# Configure different routes based on path

Lets say you want to route `/` traffic to one container (frontend) and `/api` to another container (backend). Make sure you have two components defined in `chart/values.yaml` for this or atleast two ports defined in one component:
```yaml
components:
- name: frontend
  containers:
  - image: dscr.io/youruser/frontend
  service:
    name: frontend
    ports:
    - externalPort: 80
      containerPort: 3000 
- name: backend
  containers:
  - image: dscr.io/youruser/backend
  service:
    name: backend
    ports:
    - externalPort: 80
      containerPort: 8080 
```

After you changed your `chart/values.yaml` like this, create a new file `ingress.yaml` in `chart/templates/custom` with the following content:
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: my-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  # A domain that is connected to your space!
  - host: mydomain.com
    http:
      paths:
      - path: /
        backend:
          serviceName: frontend
          servicePort: 80
      - path: /api
        backend:
          serviceName: backend
          servicePort: 80
  tls:
  - hosts:
    # Same domain name as above, this part is needed for tls
    - mydomain.com
    secretName: ingress-tls-secret
```

Make sure all other ingresses are removed in the space, so routing does not somehow collide with `kubectl delete ingress --all`. Now just run `devspace deploy` and routing should work as expected.

For more information about how you can configure ingresses, see the [kuberentes ingress documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/)

# Configure different routes based on hostname

Lets say you want to route `mydomain.com` traffic to one container (frontend) and `api.mydomain.com` to another container (backend). Make sure you have two components defined in `chart/values.yaml` for this or atleast two ports defined in one component:
```yaml
components:
- name: frontend
  containers:
  - image: dscr.io/youruser/frontend
  service:
    name: frontend
    ports:
    - externalPort: 80
      containerPort: 3000 
- name: backend
  containers:
  - image: dscr.io/youruser/backend
  service:
    name: backend
    ports:
    - externalPort: 80
      containerPort: 8080 
```

After you changed your `chart/values.yaml` like this, create a new file `ingress.yaml` in `chart/templates/custom` with the following content:
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: my-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  # A domain that is connected to your space!
  - host: mydomain.com
    http:
      paths:
      - backend:
          serviceName: frontend
          servicePort: 80
  # Another domain that is connected to your space!
  - host: api.mydomain.com
    http:
      paths:
      - backend:
          serviceName: backend
          servicePort: 80
  tls:
  - hosts:
    # Same domain name as above, this part is needed for tls
    - mydomain.com
    - api.mydomain.com
    secretName: ingress-tls-secret
```

Make sure all other ingresses are removed in the space, so routing does not somehow collide with `kubectl delete ingress --all`. Now just run `devspace deploy` and routing should work as expected.

For more information about how you can configure ingresses, see the [kuberentes ingress documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/)
