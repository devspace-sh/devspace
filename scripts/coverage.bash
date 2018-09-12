#!/usr/bin/env bash
# Copyright 2017 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
#
# This script will generate coverage.txt
set -e

# Test if we can build the program
go build main.go

PKGS=$(go list ./... | grep -v /vendor/)
for pkg in $PKGS; do
  GORACE="history_size=7" go test -race -coverprofile=profile.out -covermode=atomic $pkg
  if [[ -f profile.out ]]; then
    cat profile.out >> coverage.txt
    rm profile.out
  fi
done
