---
title: How To Connect Domains Using Ingresses
sidebar_label: Connect Domains (via Ingresses)
---

The easiest way to connect a domain to your deployment is to run the following command within your project directory:
```bash
devspace open
```
Choose **via domain** to connect a domain. After choosing this option, there are two options:

## 1. If you are using DevSpace Cloud,...
DevSpace will automatically provide a subdomain for you. 

If you want to connect a custom domain (not the auto-generated subdomain), you need to add the domain via the UI of DevSpace Cloud:
1. Go to "Spaces" (e.g. via [https://app.devspace.cloud/spaces](https://app.devspace.cloud/spaces))
2. Select the Space, you want to connect the domain to.
3. Open the "Network" tab.
4. Click on the "Create Ingress" button.
5. Specify an ingress name (e.g. "my-ingress").
6. Enter your domain as "Hostname".
7. If the message "Click here to verify the hostname." occurs, click on the link and follow the steps to verify your hostname.
8. Enter "/" or any other path as "Host Path".
9.  Click the "Create Ingress" button.

## 2. If you are **not** using DevSpace Cloud,... 
DevSpace will ask you to enter a domain name and tell you how to configure the DNS records for this domain manually.

<br>

## Troubleshooting
Here are some steps to debug issues with your application when your domain is not able to reach your application.

### Listen On All Interfaces
Make sure your application is running on `0.0.0.0` and not on `localhost`. If you see a log message in your container logs such as `Listening on localhost:PORT` or  `Listening on 127.0.0.1:PORT`, you need to change the configuration of your application, so that it starts listening on `0.0.0.0` instead of `localhost`/`127.0.0.1`.

> This is often the problem when you are able to use `devspace open` with `via localhost` but not with the `via domain` option.

### Check For Failing Containers
Make sure none of your containers is constantly restarting (`Restarts` > 0) or unable to starting (`Status` != Running):
```bash
kubectl get po
```

### Check Services and Endpoints
Make sure you have at least one service for your main application configured:
```bash
kubectl get svc
```

And make sure all your services have at least one endpoint:
```bash
kubectl get ep
```
