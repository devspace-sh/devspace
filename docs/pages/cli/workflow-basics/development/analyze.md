---
title: Analyze issues
---

DevSpace can automatically identify and analyze potential issues with your deployments:
```bash
devspace analyze
```
Running `devspace analyze` will show a lot of useful debugging information if there is an issue found, including:
- Containers that are not starting due to failed image pulling
- Containers that are not starting due to issues with the entrypoint command
- Network issues related to unhealthy pods and missing endpoints for services

## Exemplary output of `devspace analyze`
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
