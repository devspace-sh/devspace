#!/bin/sh
verbose=0
evil=1
logFile=$(mktemp)

cat /dev/null > "$logFile"
log () {
    if [ $verbose != 0 ]; then
        echo "$0 ($$): $(date +"%Y%m%d-%T") : [$1] : $2" >&1
        echo "$0 ($$): $(date +"%Y%m%d-%T") : [$1] : $2(STDERR)" >&2
        echo "$0 ($$): $(date +"%Y%m%d-%T") : [$1] : $2" >> "$logFile"
    fi
}
log "Info" "logFiles is $logFile"

log "Info" "Argiuments given: $*"

signalEXITFunction () {
    log "error" "EXIT signal received."
    log "debug" "cleaning up."
    rm -rf "$logFile"
}
singalHUPFunction () {
    log "error" "HUP signal received."
    log "info" "Configuration reloaded."
}
signalTERMFunction () {
    log "error" "TERM signal received."
    [ "$evil" -ne 1 ] && exit 1
}
signalINTFunction () {
    log "error" "INT signal received."
    exit 1
}
trap signalEXITFunction EXIT
trap singalHUPFunction HUP
trap signalTERMFunction TERM
trap signalINTFunction INT


log "Debug" "entering loop."
while true; do
    log "Debug" "sleeping for 1s."
    sleep 1
done

rm "$logFile"