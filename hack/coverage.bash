#!/usr/bin/env bash

# Set required go flags
export GO111MODULE=on
export GOFLAGS=-mod=vendor

# Test if we can build the program
echo "Building devspace..."
go build main.go || exit 1

# List packages
PKGS=$(go list ./... | grep -v /vendor/ | grep -v /examples/ | grep -v /e2e)

fail=false
for pkg in $PKGS; do
 go test -timeout 30m -race -coverprofile=profile.out -covermode=atomic $pkg
 if [ $? -ne 0 ]; then
   fail=true
 fi

 if [[ -f profile.out ]]; then
   cat profile.out >> coverage.txt
   rm profile.out
 fi
done

if [ "$fail" = true ]; then
 echo "Failure"
 exit 1
fi
