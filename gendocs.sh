#!/bin/sh

set -o errexit
set -o nounset

go run ./docs/hack/cli/main.go
go run ./docs/hack/config/partials/main.go
go run ./docs/hack/config/schemas/main.go
go run ./docs/hack/functions/main.go
