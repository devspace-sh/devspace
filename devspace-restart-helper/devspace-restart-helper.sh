#!/bin/sh
#
# A process wrapper script to simulate a container restart. This file was injected with devspace during the build process
#
# Signals processessed by the script.
# 2) SIGINT
# 15) SIGTERM
# Notes:
# - 
IFS=" "
version="v0.1"

usage () {
  cat <<USAGE
Usage: 
  $(basename "$0") [OPTIONS] CMD [ARG...]
  Options:
  --version : Display version and exit.
  --verbose : a more verbose output stream to understand internals, generally used when debugging and/or developing.
  --debug : Enable debug output, Like verbose it is generally used when debugging and/or developing.
  --development : Enable verbose, debug and development.
  --log-to-file : Enabling logging to file.
  --grace-period 7 : Grace period -in seconds- to wait for a process to exit after sending it's STOPSIGNAL, here we have several processes command, screen (if enabled), and some childeren of ours. The gracePeriod will be applied as one for each process. It is recommended to use 1/4, 1/5 of the terminationGracePeriodSeconds (default 30 seconds) value.
  --stop-signal-for-process 15 : Which signal should be send to the process(and any forked process) for graceful termination. (by default SIGTERM)
USAGE
}

parseArguments () {
  while [ $# -gt 0 ]; do
    case "$1" in
      --)
        shift
      ;;
      --version)
        shift; echo "$version";
        exit 0
      ;;
      --verbose)
        shift; verboseInput=1
      ;;
      --debug)
        shift; verboseInput=1; debugInput=1
      ;;
      --development)
        shift; verboseInput=1; debugInput=1; developmentInput=1
      ;;
      --log-to-file)
        shift; logToFileInput=1
      ;;
      --grace-period)
        shift; [ "$1" ] && gracePeriodInput="$1"
        shift
      ;;
      --stop-signal-for-process)
        shift; [ "$1" ] && stopSignalForProcessInput="$1"
        shift
      ;;
      *)
        commandAndArguments="$*"
        shift $#
      ;;
    esac
  done
  verboseInput=1; debugInput=1; developmentInput=1
  echo "commandAndArguments=$commandAndArguments"
}

init () {
  set -e
  set -u
  if [ "$debug" -eq 1 ] || [ "$development" -eq 1 ]; then
    # set -x
    set -v
  fi
  
  if [ ! -d "$workDir" ]; then
    mkdir -p "$workDir";
    log "Debug" "workDir: $workDir not found, created."
  fi
  
  touch "$logFile"

  log "Info" "My ($0) processID is $myProcessID"
  if [ "$myProcessID" -eq 1 ]; then
    log "Info" "I am the init process."
  fi
  
  userName=$( "id" "-u" "-n" )
  userGroup=$( "id" "-g" "-n" )
  log "Info" "I am running as $userName(user)/$userGroup(group)."

  echo "$myProcessID" > "$myProcessIDFile"
}

checkRequirements () {
  requiredBinaries="date cat grep"
  optionalBinaries="screen"
  requiredShellCommands="command"

  # Check if required binaries exists in the system.
  for requiredBinaryName in $requiredBinaries; do
    if ! isBinary "$requiredBinaryName"; then
      log "Fatal" "Required binary ($requiredBinaryName) doesn't exists.\n Can not continue."
    fi
  done
  
  # Check if required commands exists in the system.
  for requiredShellCommandName in $requiredShellCommands; do
    if ! isCommand "$requiredShellCommandName"; then
      log "Fatal" "Required shell command ($requiredShellCommandName) doesn't exists."
    fi
  done

  # Check if optional binaries exists in the system.
  for optionalBinaryName in $optionalBinaries; do
    if ! isBinary "$optionalBinaryName"; then
      log "Warning" "Optional shell command ($optionalBinaryName) doesn't exists."
    fi
  done
}

isCommand () {
  name="$1"
  output=$( command -v "$name")
  commandStatusCode=$?
  if [ $commandStatusCode = 0 ] && ! echo "$output" | grep -q "/" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

isBinary () {
  name="$1"
  output=$( command -v "$name")
  commandStatusCode=$?
  if [ $commandStatusCode = 0 ] && echo "$output" | grep -q "/" >/dev/null 2>&1; then
      return 0
  fi
  return 1
}

cleanup () {
  log "Debug" "Cleaning up temporary files."
  rm -rf "$myProcessIDFile"
  rm -rf "$logFile"
  rm -rf "$screenLogFile"
  rm -rf "$screenProcessIDFile"
  rm -rf "$cmdProcessIDFile"

  # Remove working directory if empty.
  if [ -d "$workDir" ]; then
    rm -r "$workDir"
  fi
}

# isZombie checks a given PID's status to determine if it is a zombie.
isZombie () {
  pid=${1?"PID is not given to the function."}
  if [ -d "/proc/$pid" ] && [ -f "/proc/$pid/status" ]; then
    # shellcheck disable=SC2002
    pidKernelState=$(cat /proc/"$pid"/status | grep -i state | awk '{ print $2 }' | tr '[:lower:]' '[:upper:]')
    [ "$pidKernelState" = "Z" ] && return 0
    return 1
  fi
  return 1
}

# isAlive checks a given PID's status to determine if it is alive.
isAlive () {
  pid=${1?"PID is not given to the function."}
  if [ -d "/proc/$pid" ] && [ -f "/proc/$pid/status" ]; then
    # shellcheck disable=SC2002
    pidKernelState=$(cat /proc/"$pid"/status | grep -i state | awk '{ print $2 }' | tr '[:lower:]' '[:upper:]')
    [ "$pidKernelState" != "Z" ] && return 0
    return 1
  fi
  return 1
}

# waitForPID, reliably (in procfs based systems) waits the process to exit.
# It allows limit the waiting period by using the maxWait parameter. (like timeout)
# This function can be used while waiting / expecting signals, unlike long running processes.
waitForPID () {
  pid=${1?"PID is not given to the function."}
  maxWait=${2:-0}
  startToWait=$( date +%s )
  elapsedTime=0
  while isAlive "$pid" || isZombie "$pid"; do
    currentTime=$( date +%s )
    elapsedTime=$(( currentTime-startToWait ))
    if [ "$maxWait" != 0 ] && [ "$elapsedTime" -ge "$maxWait" ]; then log "Debug" "Process ($pid) exceeded maxWait($maxWait)"; return 1; fi
    sleep "$waitForPIDLoopDuration"
  done
  return 0
}

childProcessReaper () {
  sh -c "wait &"
}

logPrint () {
  message=$1
  logLine="($$)$(date +"$dateFormat") : $message"
  echo "$logLine"
  if [ "$logToFile" ] && [ -f "$logFile" ]; then
    echo "$logLine" >> "$logFile"
  fi
}

log () {
  severityInput=$1
  severity="$(echo "$severityInput" | tr '[:lower:]' '[:upper:]')"
  message=$2
  logLine="[$severity] : $message"
  case "$severity" in
    FATAL)
      logPrint "$logLine"
      exit 1
    ;;
    ERROR)
      logPrint "$logLine"
    ;;
    WARNING)
      logPrint "$logLine"
    ;;
    INFO)
      if [ "$verbose" -eq 1 ] || [ "$debug" -eq 1 ] || [ "$development" -eq 1 ]; then
        logPrint "$logLine"
      fi
    ;;
    DEBUG)
      if [ "$debug" -eq 1 ] || [ "$development" -eq 1 ]; then
        logPrint "$logLine"
      fi
    ;;
    DEVELOPMENT)
      if [ "$development" -eq 1 ]; then
        logPrint "$logLine"
      fi
    ;;
    *)
      if [ "$development" -eq 1 ]; then
        logPrint "$logLine"
      fi
    ;;
  esac
}



checkAndInstallScreen() {
  log "Debug" "checkAndInstallScreen called."
  if ! command -v screen >/dev/null 2>&1 && $installScreen; then
    log "Info" "Screen not found and, we are asked to install, installing."
    if command -v apk >/dev/null 2>&1; then
      apk add --no-cache screen >/dev/null 2>&1
      lastCommandsExitStatus=$?
      if [ ! $lastCommandsExitStatus ]; then
        log "Error" "Couldn't install screen using apk"
      fi
    elif command -v apt-get >/dev/null 2>&1; then
      export DEBIAN_FRONTEND=noninteractive
      apt-get -qq update >/dev/null 2>&1 && apt-get install -y screen  >/dev/null 2>&1 && rm -rf /var/lib/apt/lists/* >/dev/null 2>&1
      lastCommandsExitStatus=$?
      if [ ! $lastCommandsExitStatus ]; then
        log "Error" "Couldn't install screen using apt-get"
      fi
    else
      log "Error" "Couldn't install screen using neither apt-get nor apk."
    fi
  fi
  if command -v screen >/dev/null 2>&1; then
    log "Info" "Screen installed successfully."
  else
    log "Info" "Coudln't find screen, need to fallback."
  fi
}



# getProcessIDFromFile reads a pid file and sends the value to stdout
getProcessIDFromFile () {
  processIDFile=${1?"processIDFile variable is not defined."}
  [ -f "$processIDFile" ] && cat "$processIDFile" && return 0
  return 1
}

signalProcessGroupByProcessGroupIDFile () {
  pgidFile=${1?"pgidFile variable is not defined."}
  signal=${2?"signal variable is not defined"}
  pgidToSignal="$( getProcessIDFromFile "$pgidFile" )"
  [ "$pgidToSignal" ] && sendSignalToProcessOrProcessGroup "$pgidToSignal" "$signal" true
  return $?
}

# isProcessSignalable do not actually sends a signal but rather enables error handling, so if successful we can signal to pid/pgid (PID exists and we have permission).
# A non-zero exit code doesn't mean the process doesn't exists, rather we are not able to signal whatever the reason might be.
# This can be trying to signal a process that is not in our namespace.
isProcessSignalable () {
  processID=${1?"processID variable is not defined."}
  signal=0
  if sendSignalToProcessOrProcessGroup "$processID" "$signal"; then
    return 0
  fi
  lastCommandsExitStatus=$?
  return $lastCommandsExitStatus
}

sendSignalToProcessOrProcessGroup () {
  processOrProcessGroupID=${1?"processOrProcessGroupID variable is not defined."}
  signal=${2?"signal variable is not defined."}
  groupInput=${3:-"FALSE"}
  group="$(echo "$groupInput" | tr '[:lower:]' '[:upper:]')"
  if [ "$group" = "TRUE" ]; then
    processOrProcessGroupID=$((0-processOrProcessGroupID))
  fi
  signal=$((0-signal))
  log "Debug" "Signalling ($signal) to PID/PGID ($processOrProcessGroupID) --group ($group)"
  if [ -x "/bin/kill" ]; then
    output=$(/bin/kill "$signal" "$processOrProcessGroupID")
  else
    output=$(kill "$signal" "$processOrProcessGroupID")
  fi
  lastCommandsExitStatus=$?
  if [ $lastCommandsExitStatus -eq 0 ]; then
    log "Debug" "Signal sent successfully (\$?=$lastCommandsExitStatus)"
    return 0
  else
    log "Debug" "Can not signal ($signal) to PID/PGID ($processOrProcessGroupID) --group ($group), it doesn't exists or we don't have permission."
    log "Development" "Kill command output=$output"
    log "Development" "lastCommandsExitStatus=$lastCommandsExitStatus"
    return 1
  fi
}

killProcessGroupGracefullyByProcessIDFile () {
  pidFile=${1?"pidFile variable is not defined."}
  stopSignal=${2:-$stopSignalDefault}
  signal="$stopSignal"
  pidToKill="$( getProcessIDFromFile "$pidFile" )"
  if [ "$pidToKill" ]; then
    sendSignalToProcessOrProcessGroup "$pidToKill" "$signal" false
    sleep 0.05
    log "Debug" "Waiting for a maximum of ($gracePeriod) seconds for PID ($pidToKill) to exit gracefully."
    waitForPID "$pidToKill" "$gracePeriod"
    # NOTE: At this point either process with id pidToKill died OR it has elapsed more than $gracePeriod.
    # waitForPID doesn't kill any process. Check if alive:
    isAlive "$pidToKill"
    lastCommandsExitStatus=$?
    if [ $lastCommandsExitStatus -eq 0 ]; then
      log "Debug" "PID ($pidToKill) didn't exit within ($gracePeriod) seconds, signalling SIGKILL."
      signal="9"
      sendSignalToProcessOrProcessGroup "$pidToKill" "$signal" true
      sleep 0.05
    else
      log "Debug" "PID ($pidToKill) exited gracefully"
    fi  
  else
    log "Error" "killProcessGroupGracefullyByProcessIDFile called with pidFile=($1), but file content is empty."
  fi
}


quit() {
  restart=false
  screenProcessID="$( getProcessIDFromFile "$screenProcessIDFile" )"
  cmdProcessID="$( getProcessIDFromFile "$cmdProcessIDFile" )"
  log "Info" "Trying to kill command process ($cmdProcessID) gracefully."
  # $stopSignalForProcess can be overridden in the input values.
  killProcessGroupGracefullyByProcessIDFile "$cmdProcessIDFile" "$stopSignalForProcess"
  
  # Check if Screen was ever used and screen process is alive.
  isAlive "$screenProcessID"
  lastCommandsExitStatus=$?
  if [ "$lastCommandsExitStatus" -eq 0 ]; then
    log "Debug" "Screen seems to be alive."
    log "Info" "Trying to kill Screen process ($screenProcessID) gracefully."
    killProcessGroupGracefullyByProcessIDFile "$screenProcessIDFile" "$stopSignalDefault"
  fi
  # Reap zombies and left children.
  childProcessReaper
  
  # If we are init let's signal everybody in the namespace
  if [ "$myProcessID" -eq 1 ]; then
    log "Warning" "I am acting init, safe to kill all processes in the namespace."
    sendSignalToProcessOrProcessGroup "-1" "9" false
    sleep 0.25
    # sendSignalToProcessOrProcessGroup "-1" "2" false
  fi
}

restartLoop () {
  while $restart; do
  set +e
  
  if command -v screen >/dev/null; then
    log "Debug" "Screen found using screen."
    rm -rf "$screenLogFile"
    rm -rf "$screenProcessIDFile"
    
    log "Info" "Command process is about to start."
    # Use: screen [-opts] [cmd [args]]
    # -L   tells screen to turn on automatic output logging for the windows.
    # -dmS name     Start as daemon: Screen session in detached mode
    # The command and arguments for the screen to run is a shell snippet so we can gather info.
    # However the shell would not process the signals and we would loose the control so using exec so that the shell will be replaced with the command process.
    setsid screen -L -Logfile "$screenLogFile" -dmS "$screenSessionName" sh -c "echo \$$>$cmdProcessIDFile; echo \$PPID>$screenProcessIDFile; exec $commandAndArguments; exit;" &
    sleep "$timeToWaitForScreenAndCommandProceses"

    screenProcessID="$( getProcessIDFromFile "$screenProcessIDFile" )"
    cmdProcessID="$( getProcessIDFromFile "$cmdProcessIDFile" )"
    isAlive "$screenProcessID" && log "Development" "Screen is alive."
    log "Info" "Screen PID: $screenProcessID | Command PID: $cmdProcessID"
    
    # Use: screen [-opts] [cmd [args]]
    # -S sessionname
    # -Q   Some commands now can be queried from a remote session using this flag, e.g. "screen -Q windows". The commands will send the response to the stdout of the querying process. If there was an error in the command, then the querying process will exit with a non-zero status.
    iteration=0
    while [ ! -f "$screenProcessIDFile" ] || [ ! -f "$screenLogFile" ] || ! screen -S "$screenProcessID.$screenSessionName" -Q select . >/dev/null; do
      sleep 0.3
      iteration=$((iteration + 1))
      if [ $iteration -gt 25 ]; then
        [ -f "$screenProcessIDFile" ] && log "Development" "$screenProcessIDFile doesn't exists."
        [ -f "$screenLogFile" ] || log "Development" "$screenLogFile doesn't exists."
        screen -S "$screenProcessID.$screenSessionName" -Q select . >/dev/null || log "Development" "Sending command to main Screen session failed, with ($?)."
        ! isAlive "$screenProcessID" && log "Development" "Screen is not alive."
        log "Fatal" "Loop iteration exceeded maximum number of iterations."
      fi
    done

    # Use: screen [-opts] [cmd [args]]
    # -r            Reattach to a detached screen process.
    # -X            Execute <cmd> as a screen command in the specified session.
    screen -r "$screenProcessID.$screenSessionName" -X colon "logfile flush 1^M"
    
    # Start reading the log file
    tail -f "$screenLogFile" &

    # This loop needs to give control back to shell so that shell can trap.
    while waitForPID "$cmdProcessID"; do
      sleep 10;
      childProcessReaper
    done
    log "Warning" "############### Process exited ###############"

  else
    log "Debug" "Screen not found fallback to setsid w/o screen."
    log "Info" "Command process is about to start."
    sh -c "setsid exec $*" &
    cmdProcessID=$!
    echo "$cmdProcessID" > "$cmdProcessIDFile"
    log "Info" "Command PID: $cmdProcessID"

    # This loop needs to give control back to shell so that shell can trap.
    while waitForPID "$cmdProcessID"; do 
      sleep 10;
      childProcessReaper
    done
    log "Warning" "############### Process exited ###############"
  fi
  set -e

  if $restart; then
    rm -rf "$cmdProcessIDFile"
    log "Warning" "############### Restarting ###############"
    sleep 7
  fi
done
}
######################################################################
# Internal variables
workDir="/tmp/.devspace"
logFile="$workDir/devspace-restart-helper.log"
dateFormat="%Y%m%d-%T" # Datetime format to ber used in logging.
screenSessionName="devspace-restart-helper"
screenProcessID=""
screenProcessIDFile="$workDir/screen.pid"
screenLogFile="$workDir/screen.log"
myProcessID=$$
myProcessIDFile="$workDir/devspace-restart-helper.pid"
cmdProcessID=""
cmdProcessIDFile="$workDir/cmd.pid"
waitForPIDLoopDuration=1
stopSignalDefault=15
installScreen=true # By default if screen is not installed we try to install usiong either apk or apt-get. Set to false to disable.
timeToWaitForScreenAndCommandProceses=5
# Variables and their defaults that can be set with arguments
verbose=${verboseInput:-1} # Verbose logging is enabled by default.
debug=${debugInput:-0} # Debug logging is disabled by default.
development=${developmentInput:-0} # Development logging is disabled by default.
stopSignalForProcess=${stopSignalForProcessInput:-$stopSignalDefault}
gracePeriod=${gracePeriodInput:-7} # Grace period (in seconds) that we wait for the process to exit, after sending TERM signal to it. K8s default grace period is 30 seconds. We allow less so that everybody have time to cleanup.
logToFile=${logToFileInput:-1}

######################################################################
trap processSignalEXIT EXIT
processSignalEXIT () {
  signal="EXIT"
  if [ "$development" -ne 1 ]; then
    cleanup
  fi
}

trap processSignalTERM TERM
processSignalTERM () {
  # Make sure to ignore the signals we recently registered from here on.
  trap '' TERM INT
  signal="15"
  log "Warning" "Signal ($signal) received."
  quit
  exit 143
}

trap processSignalINT INT
processSignalINT () {
  # Make sure to ignore the signals we recently registered from here on.
  trap '' TERM INT
  signal="2"
  log "Warning" "Signal ($signal) received."
  quit
  exit 130
}

######################################################################
main () {
  if [ ! $# -ge 1 ]; then
    log "Error" "($#) argument(s) supplied. At least one argument (command to run) is required."
    usage
    exit 1
  fi
  parseArguments "$@"
  init
  checkRequirements
  checkAndInstallScreen
  restart=true
  restartLoop
}
######################################################################
main "$@"
