---
title: 4. Debug with DevSpace
id: version-v3.5.18-debugging
original_id: debugging
---

DevSpace CLI provides useful features for debugging applications that are running inside a Kubernetes clusters.

## Analyze issues in your namespaces
DevSpace CLI can automatically analyze Kubernetes namespaces and identify potential issues with your deployments. Simply run this command to let DevSpace CLI look for a variety of issues:
```bash
devspace analyze
```

Running `devspace analyze` will show a lot of useful debugging information, including:
- Containers that are not starting due to failed image pulling
- Containers that are not starting due to issues with the entrypoint command
- Network issues related to unhealthy pods and missing endpoints for services

<details>
<summary>
### Show an exemplary output of `devspace analyze`
</summary>
```bash
$ devspace analyze
[info]   Loaded config from devspace-configs.yaml
                                           
  ================================================================================
                            Pods (1 potential issue(s))                           
  ================================================================================
  Pod default-59bd65f686-h7n9r:  
    Status: Running  
    Created: 22s ago  
    Container: 1/1 running  
    Problems:   
      - Container: container-0  
        Restarts: 1  
        Last Restart: 8s ago  
        Last Exit: Error (Code: 1)  
        Last Execution Log: 

> my-app@0.0.1 start /app
> node index.js

Example app listening on port 3000!
/app/index.js:14
  test();
  ^

ReferenceError: test is not defined
    at Timeout.setTimeout [as _onTimeout] (/app/index.js:14:3)
    at ontimeout (timers.js:498:11)
    at tryOnTimeout (timers.js:323:5)
    at Timer.listOnTimeout (timers.js:290:5)
npm ERR! code ELIFECYCLE
npm ERR! errno 1
npm ERR! my-app@0.0.1 start: `node index.js`
npm ERR! Exit status 1
npm ERR! 
npm ERR! Failed at the my-app@0.0.1 start script.
npm ERR! This is probably not a problem with npm. There is likely additional logging output above.

npm ERR! A complete log of this run can be found in:
npm ERR!     /root/.npm/_logs/2019-03-19T22_51_03_656Z-debug.log
```
</details>

## Debug applications with remote debuggers
DevSpace CLI lets you easily [start applications in development mode](../getting-started/development) and connect remote debuggers for your application using the following steps:
1. Configure DevSpace CLI to [use a development Dockerfile](../development/overrides#configuring-a-different-dockerfile-during-devspace-dev) that:
   - ships with the appropriate tools for debugging your application
   - starts your application together with the debugger, e.g. setting the `ENTRYPOINT` of your Dockerfile to `node --inspect=0.0.0.0:9229 index.js` would start the Node.js remote debugger on port `9229`
2. Define port-forwarding for the port of your remote debugger (e.g. `9229`) within the `dev.ports` section of your `devspace.yaml`
3. Connect your IDE to the remote debugger (see the docs of your IDE for help)
4. Set breakpoints and debug your application directly inside Kubernetes
