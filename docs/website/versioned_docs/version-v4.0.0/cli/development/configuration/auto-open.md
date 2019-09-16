---
title: Configuring Automatic Opening of URLs
sidebar_label: Auto-Open
id: version-v4.0.0-auto-open
original_id: auto-open
---


DevSpace allows you to define URLs that should open after deploying an application in development mode, i.e. using `devspace dev`.

The configuration for automatically opening URLs can be found in the `dev.open` section of `devspace.yaml`.
```yaml
dev:
  open:
  - url: http://localhost:3000/login
```

> Setting `dev.open` only affects `devspace dev`. To open your application after running `devspace deploy`, run `devspace open`.


## `dev.open`
The `open` option expects an array of auto-open configurations with exactly one of the following properties:
- `url` to open a URL in the default browser

> Providing a URL results in the following behavior during `devspace dev`:
> - DevSpace deploys the application and starts to periodically send `HTTP GET` requests to the provideded `url`.
> - As soon as the first HTTP response has a status code which is neither 502 (Bad Gateway) nor 503 (Service Unavailable), DevSpace assumes that the application is now started, stops sending any further requests and opens the provided URL in the browser.

#### Default Value For `open`
```yaml
open: []
```

#### Example: Open URL when Starting Dev Mode
```yaml
dev:
  open:
  - url: http://localhost:3000/login
```
**Explanation:**  
Running `devspace dev` using the above configuration would do the following:
- Build images and deploy the application
- Start port-forwarding and code-synchronization
- DevSpace opens the browser with URL `http://localhost:3000/login`
