package helper

// RestartScriptPath is the absolute path of the restart script in the container
var RestartScriptPath = "/" + RestartScriptName

// RestartScriptName is the filename of the restart script in the container
const RestartScriptName = "devspace-restart-helper"

// RestartHelperScript is the content of the restart script in the container
const RestartHelperScript = `#!/bin/sh
#
# A process wrapper script to simulate a container restart. This file was injected with devspace during the build process
#

set -e

pid=""

trap quit TERM INT

quit() {
  if [ -n "$pid" ]; then
    kill $pid
  fi
}

while true; do
    "$@" &
    pid=$!
    echo "$pid" > /devspace-pid
    set +e
    wait $pid
    exit_code=$?
    set -e
    if [ -f /devspace-pid ]; then
        exit $exit_code
    fi
    echo "Restart container"
done
`
