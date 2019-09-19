#!/usr/bin/env bash
# Copyright 2017 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
#
# This script will generate coverage.txt
export GO111MODULE=on
export GOFLAGS=-mod=vendor
# Test if we can build the program
go build main.go || exit 1
PKGS=$(go list ./... | grep -v /vendor/ | grep -v /examples/)
fail=false
for pkg in $PKGS; do
 go test -race -coverprofile=profile.out -covermode=atomic $pkg
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
