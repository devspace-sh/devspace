---
title: SSL certificates (TLS)
---

In DevSpace cloud, SSL certificates for custom domains are automatically generated with the help of the [cert-manager](https://github.com/jetstack/cert-manager). Behind the scenes a [lets encrypt](https://letsencrypt.org/) certificate will be created for you. In order to create the certificate, it is essential that your custom domain points to the correct IP adress and that your space has enough resources left to start the cert-manager challenge resolver pod.

## Make sure the space has enough resources to start the challenge resolver pod

The challenge resolver pod needs `100m` cpu and `200Mi` memory resources. If your space has not enough resources anymore, the challenge resolver pod cannot be started and hence the certificate cannot be created. To lower the resources of your space you can edit the `chart/values.yaml` in your project and redeploy the application. You can increase the used resources after the certificate was created. 

Example `chart/values.yaml`:

```yaml
components:
- name: default
  containers:
  - image: dscr.io/myuser/devspace
    resources:
      limits:
        # Lower this
        cpu: "300m"
        # Lower this
        memory: "300Mi"
        ephemeralStorage: "1Gi"
    ...
  service:
    ...
...
```

## Make sure your domain points to the correct IP address

Goto Spaces -> Your Space Name -> Network -> Click on 'ingress' Button
![alt text](/img/ingress.png "Ingress")

Make sure you make an A record entry in your domain provider to shown address.
![alt text](/img/load-balancer-ip.png "Load Balancer Ip")
