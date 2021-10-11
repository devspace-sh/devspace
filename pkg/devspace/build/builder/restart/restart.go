package restart

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// LegacyScriptPath is the old absolute path of the restart script in the container
var LegacyScriptPath = "/" + ScriptName

// ScriptPath is the absolute path of the restart script in the container
var ScriptPath = "/.devspace/" + ScriptName

// ScriptContextPath is the absolute path of the restart script in the build context
var ScriptContextPath = "/.devspace/.devspace/" + ScriptName

// ScriptName is the filename of the restart script in the container
const ScriptName = "devspace-restart-helper"

// LegacyProcessIDFilePath is the old path where the current active process id is stored
const LegacyProcessIDFilePath = "/devspace-pid"

// ProcessIDFilePath is the path where the current active process id is stored
const ProcessIDFilePath = "/.devspace/devspace-pid"

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

if [ "$DEVSPACE_MANUAL_START" = "true" ]; then
  setsid sleep 999999 &
  pid=$!
  echo "$pid" > /.devspace/devspace-pid
  set +e
  wait $pid
  set -e
  unset DEVSPACE_MANUAL_START
  printf "\n################ Start container ################\n\n"
fi

while true; do
  setsid "$@" &
  pid=$!
  echo "$pid" > /.devspace/devspace-pid
  set +e
  wait $pid
  exit_code=$?
  if [ -f /.devspace/devspace-pid ]; then
    rm -f /.devspace/devspace-pid 	
    printf "\nContainer exited with $exit_code. Will restart in 7 seconds...\n"
    sleep 7
  fi
  set -e
  printf "\n\n############### Restart container ###############\n\n"
done
`

// LoadRestartHelper loads the restart helper script from either
// a path or returns the bundled version of it.
func LoadRestartHelper(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return HelperScript, nil
	} else if isRemoteHTTP(path) {
		resp, err := http.Get(path)
		if err != nil {
			return "", err
		}

		out, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		} else if resp.StatusCode >= 400 {
			return "", fmt.Errorf("reading %s failed with code %d: %s", path, resp.StatusCode, string(out))
		}

		return string(out), nil
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// isRemoteHTTP checks if the source is a http/https url and a yaml
func isRemoteHTTP(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
