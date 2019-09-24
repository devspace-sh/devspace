#!/usr/bin/env bash
# This script will build devspace and calculate hash for each
# (DEVSPACE_BUILD_PLATFORMS, DEVSPACE_BUILD_ARCHS) pair.
# DEVSPACE_BUILD_PLATFORMS="linux" DEVSPACE_BUILD_ARCHS="amd64" ./hack/build-all.bash
# can be called to build only for linux-amd64

set -e

export GO111MODULE=on
export GOFLAGS=-mod=vendor

# Update vendor directory
# go mod vendor

DEVSPACE_ROOT=$(git rev-parse --show-toplevel)
VERSION=$(git describe --tags $(git rev-list --tags --max-count=1))
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
DATE=$(date "+%Y-%m-%d")
BUILD_PLATFORM=$(uname -a | awk '{print tolower($1);}')
IMPORT_DURING_SOLVE=${IMPORT_DURING_SOLVE:-false}

if [[ "$(pwd)" != "${DEVSPACE_ROOT}" ]]; then
  echo "you are not in the root of the repo" 1>&2
  echo "please cd to ${DEVSPACE_ROOT} before running this script" 1>&2
  exit 1
fi

GO_BUILD_CMD="go build -a -installsuffix cgo"
GO_BUILD_LDFLAGS="-s -w -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${DATE} -X main.version=${VERSION} -X github.com/devspace-cloud/devspace/pkg/util/analytics.token=${ANALYTICS_TOKEN} -X github.com/devspace-cloud/devspace/pkg/util/analytics.eventEndpoint=${ANALYTICS_ENDPOINT_EVENT} -X github.com/devspace-cloud/devspace/pkg/util/analytics.userEndpoint=${ANALYTICS_ENDPOINT_USER} -X main.flagImportDuringSolve=${IMPORT_DURING_SOLVE}"

if [[ -z "${DEVSPACE_BUILD_PLATFORMS}" ]]; then
    DEVSPACE_BUILD_PLATFORMS="linux windows darwin"
fi

if [[ -z "${DEVSPACE_BUILD_ARCHS}" ]]; then
    DEVSPACE_BUILD_ARCHS="amd64 386 ppc64 ppc64le"
fi

mkdir -p "${DEVSPACE_ROOT}/release"

for OS in ${DEVSPACE_BUILD_PLATFORMS[@]}; do
  for ARCH in ${DEVSPACE_BUILD_ARCHS[@]}; do
    NAME="devspace-${OS}-${ARCH}"
    if [[ "${OS}" == "windows" ]]; then
      NAME="${NAME}.exe"
    fi

    # Enable CGO if building for OS X on OS X; see
    # https://github.com/golang/dep/issues/1838 for details.
    if [[ "${OS}" == "darwin" && "${BUILD_PLATFORM}" == "darwin" ]]; then
      CGO_ENABLED=1
    else
      CGO_ENABLED=0
    fi

    if [[ "${ARCH}" == "ppc64" || "${ARCH}" == "ppc64le" ]] && [[ "${OS}" != "linux" ]]; then
        # ppc64 and ppc64le are only supported on Linux.
        echo "Building for ${OS}/${ARCH} not supported."
    else
        echo "Building for ${OS}/${ARCH} with CGO_ENABLED=${CGO_ENABLED}"
        GOARCH=${ARCH} GOOS=${OS} CGO_ENABLED=${CGO_ENABLED} ${GO_BUILD_CMD} -ldflags "${GO_BUILD_LDFLAGS}"\
            -o "${DEVSPACE_ROOT}/release/${NAME}" .
        shasum -a 256 "${DEVSPACE_ROOT}/release/${NAME}" > "${DEVSPACE_ROOT}/release/${NAME}".sha256
    fi
  done
done

# build sync helper
echo "Building sync helper"
GOARCH=386 GOOS=linux go build -ldflags "-s -w -X main.version=${VERSION}" -o "${DEVSPACE_ROOT}/release/sync" sync/stub/main.go
shasum -a 256 "${DEVSPACE_ROOT}/release/sync" > "${DEVSPACE_ROOT}/release/sync".sha256
