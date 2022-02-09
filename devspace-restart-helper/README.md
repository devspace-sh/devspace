# devspace-restart-helper

Documentation for the concept and how it works:

[Container Restart Helper](https://devspace.sh/cli/docs/configuration/images/inject-restart-helper)

```shell
Usage: 
  devspace-restart-helper.sh [OPTIONS] CMD [ARG...]
  Options:
  --version : Display version and exit.
  --verbose : a more verbose output stream to understand internals, generally used when debugging and/or developing.
  --debug : Enable debug output, Like verbose it is generally used when debugging and/or developing.
  --development : Enable verbose, debug and development.
  --log-to-file : Enabling logging to file.
  --grace-period 7 : Grace period -in seconds- to wait for a process to exit after sending it's STOPSIGNAL, here we have several processes command, screen (if enabled), and some childeren of ours. The gracePeriod will be applied as one for each process. It is recommended to use 1/4, 1/5 of the terminationGracePeriodSeconds (default 30 seconds) value.
  --stop-signal-for-process 15 : Which signal should be send to the process(and any forked process) for graceful termination. (by default SIGTERM)
```
