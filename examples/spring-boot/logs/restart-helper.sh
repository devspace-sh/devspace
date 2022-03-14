#!/bin/sh
set -e
pid=""
trap quit TERM INT
quit() {
  if [ -n "$pid" ]; then
    kill $pid
  fi
}

while true; do
  printf "\n\n############### Delete target/.jar files ###############\n\n"
  rm -rfv target/*.jar
  printf "\n\n############### Compile packages ###############\n\n"
  ./mvnw package
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
