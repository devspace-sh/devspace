package restart

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// LegacyScriptPath is the old absolute path of the restart script in the container
var LegacyScriptPath = "/" + ScriptName

// ScriptPath is the absolute path of the restart script in the container
var ScriptPath = "/.devspace/" + ScriptName

// TouchPath is the absolute path of the touch file that signals initial syncing is done
// and the container can start
var TouchPath = "/.devspace/start"

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
# DevSpace Restart Helper
#
set -e

restart=true
screenSessionName="devspace"
workDir="$PWD"
tmpDir="/.devspace"
screenLogFile="$tmpDir/screenlog.0"
pidFile="$tmpDir/devspace-pid"
sidFile="$tmpDir/devspace-sid"
touchFile="$tmpDir/start"

mkdir -p $tmpDir

trap quit TERM INT
quit() {
  restart=false
  if [ -f "$pidFile" ]; then
    pidToKill="$(cat $pidFile)"
    kill -2 $((0-$pidToKill)) >/dev/null 2>&1
    timeout 5 sh -c "while [ -e /proc/$pidToKill ]; do sleep 1; done"
    kill -15 $((0-$pidToKill)) >/dev/null 2>&1
    timeout 5 sh -c "while [ -e /proc/$pidToKill ]; do sleep 1; done"
    kill -9 $((0-$pidToKill)) >/dev/null 2>&1
    timeout 5 sh -c "while [ -e /proc/$pidToKill ]; do sleep 1; done"
  fi

  if [ -f "$ppidFile" ]; then
    pidToKill="$(cat $ppidFile)"
    kill -9 $((0-$pidToKill)) >/dev/null 2>&1
  fi
}

counter=0
while ! [ -f $touchFile ]; do
  if [ "$counter" = "0" ]; then
    echo "Container started with restart helper."
    echo "Waiting for initial sync to complete or file $touchFile to exist before starting the application..."
  else
    if [ "$counter" = 10 ]; then
      echo "(Still waiting...)"
      counter=0
    fi
  fi
  sleep 1
  counter=$((counter + 1))
done

if ! [ "$counter" = "0" ]; then
  echo "Starting application..."
fi

while $restart; do
  set +e
  if command -v screen >/dev/null; then
    rm -f "$screenLogFile"
    rm -f "$pidFile"
    rm -f "$sidFile"

    cd "$tmpDir"

    screen -q -L -dmS $screenSessionName sh -c 'echo $$>"'$pidFile'"; echo $PPID>"'$sidFile'"; cd "'$workDir'"; exec "$@"; exit;' _ "$@"

    while [ ! -f "$sidFile" ]; do
      sleep 0.1
    done
    sid="$(cat $sidFile).${screenSessionName}"
    pid="$(cat $pidFile)"

    screen -q -S "${sid}" -X colon "logfile flush 1^M"

    tail -f "$screenLogFile" &
    tailPid=$!
    # This is a workaround on tail --pid not supported in all
	# minimal shells
    while [ -e /proc/$pid ]; do
		# Until $pid exist, let's wait
        sleep 1
    done
    kill $tailPid
  else
    setsid "$@" &
    pid=$!
    echo "$pid" >"$pidFile"
    wait "$pid"
  fi
  set -e

  if $restart; then
    if [ -f "$pidFile" ]; then
      rm -f "$pidFile"
      printf "\nContainer exited. Will restart in 7 seconds...\n"
      sleep 7
    fi
    printf "\n\n############### Restart container ###############\n\n"
  fi
done
`

const LegacyRestartHelperScript = `#!/bin/sh
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
	return loadRestartHelper(path, HelperScript)
}

// LoadLegacyRestartHelper loads the legacy restart helper script from either
// a path or returns the bundled version of it.
func LoadLegacyRestartHelper(path string) (string, error) {
	return loadRestartHelper(path, LegacyRestartHelperScript)
}

func loadRestartHelper(path, defaultScript string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultScript, nil
	} else if isRemoteHTTP(path) {
		resp, err := http.Get(path)
		if err != nil {
			return "", err
		}

		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		} else if resp.StatusCode >= 400 {
			return "", fmt.Errorf("reading %s failed with code %d: %s", path, resp.StatusCode, string(out))
		}

		return string(out), nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// isRemoteHTTP checks if the source is a http/https url and a yaml
func isRemoteHTTP(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
