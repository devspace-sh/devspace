---
title: Health checks
---

Components allow you to use the Kubernetes feature of defining health checks:
- `livenessProbe` allows Kubernetes to check wheather the container is running correctly and restart/recreate it if necessary
- `readinessProbe` allows Kubernetes to check when the container is ready to accept requests (e.g. becoming ready after completing initial startup tasks)

```yaml
deployments:
- name: backend
  component:
    containers:
    - image: dscr.io/username/api-server
      livenessProbe:
        httpGet:
          path: /healthz
          port: 8080
          httpHeaders:
          - name: Custom-Header
            value: Awesome
        initialDelaySeconds: 3
        periodSeconds: 3
      readinessProbe:
        exec:
          command:
          - cat
          - /tmp/healthy
        initialDelaySeconds: 5
        periodSeconds: 5
```
The above example would define an [HTTP livenessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-a-liveness-http-request) and an [exec readinessProbe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-readiness-probes) for the container. Components allow you to use all capabilities for livenessProbes and readinessProbes that the Kubernetes specification provides.

For more information, please take a look at the [Kubernetes documentation for configuring liveness and readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/).
