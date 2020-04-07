package restart

// ScriptPath is the absolute path of the restart script in the container
var ScriptPath = "/" + ScriptName

// ScriptContextPath is the absolute path of the restart script in the container
var ScriptContextPath = "/.devspace/" + ScriptName

// ScriptName is the filename of the restart script in the container
const ScriptName = "devspace-restart-helper"

// ProcessIDFilePath is the path where the current active process id is stored
const ProcessIDFilePath = "/devspace-pid"

// HelperScript is the content of the restart script in the container
const HelperScript = `#!/bin/sh
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
    setsid "$@" &
    pid=$!
    echo "$pid" > /devspace-pid
    set +e
    wait $pid
    exit_code=$?
    if [ -f /devspace-pid ]; then
      # if the sync is currently active we try to restart instead of exiting
      if [ -f /tmp/sync ]; then
        rm -f /devspace-pid 	
        printf "\nContainer exited with $exit_code. Will restart in 7 seconds...\n"
        sleep 7
      else
        exit $exit_code
      fi
    fi
    set -e
    printf "\nRestart container\n"
done
`
