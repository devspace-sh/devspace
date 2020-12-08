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

if [[ "$(pwd)" != "${DEVSPACE_ROOT}" ]]; then
  echo "you are not in the root of the repo" 1>&2
  echo "please cd to ${DEVSPACE_ROOT} before running this script" 1>&2
  exit 1
fi

GO_BUILD_CMD="go build -a"
GO_BUILD_LDFLAGS="-s -w -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${DATE} -X main.version=${VERSION}"

if [[ -z "${DEVSPACE_BUILD_PLATFORMS}" ]]; then
    DEVSPACE_BUILD_PLATFORMS="linux windows darwin"
fi

if [[ -z "${DEVSPACE_BUILD_ARCHS}" ]]; then
    DEVSPACE_BUILD_ARCHS="amd64 386"
fi

# Install bin data
go get -mod= -u github.com/go-bindata/go-bindata/...

# Create the release directory
mkdir -p "${DEVSPACE_ROOT}/release"

# Move ui.tar.gz to releases
echo "Moving ui"
mv ui.tar.gz "${DEVSPACE_ROOT}/release/ui.tar.gz"
shasum -a 256 "${DEVSPACE_ROOT}/release/ui.tar.gz" > "${DEVSPACE_ROOT}/release/ui.tar.gz".sha256

# build devspace helper
echo "Building devspace helper"
GOARCH=386 GOOS=linux go build -ldflags "-s -w -X github.com/devspace-cloud/devspace/helper/cmd.version=${VERSION}" -o "${DEVSPACE_ROOT}/release/devspacehelper" helper/main.go
shasum -a 256 "${DEVSPACE_ROOT}/release/devspacehelper" > "${DEVSPACE_ROOT}/release/devspacehelper".sha256

# build bin data
cd ${DEVSPACE_ROOT} && go-bindata -o assets/assets.go -pkg assets release/devspacehelper release/ui.tar.gz

for OS in ${DEVSPACE_BUILD_PLATFORMS[@]}; do
  for ARCH in ${DEVSPACE_BUILD_ARCHS[@]}; do
    NAME="devspace-${OS}-${ARCH}"
    if [[ "${OS}" == "windows" ]]; then
      NAME="${NAME}.exe"
    fi

    # Enable CGO if building for OS X on OS X; this is required for 
    # github.com/rjeczalik/notify; see https://github.com/rjeczalik/notify/pull/182
    if [[ "${OS}" == "darwin" && "${BUILD_PLATFORM}" == "darwin" ]]; then
      CGO_ENABLED=1
    else
      CGO_ENABLED=0
    fi

    if [[ "${ARCH}" == "386" && "${OS}" == "darwin" ]]; then
        # darwin 386 is deprecated and shouldn't be used anymore
        echo "Building for ${OS}/${ARCH} not supported."
    else
        echo "Building for ${OS}/${ARCH} with CGO_ENABLED=${CGO_ENABLED}"
        GOARCH=${ARCH} GOOS=${OS} CGO_ENABLED=${CGO_ENABLED} ${GO_BUILD_CMD} -ldflags "${GO_BUILD_LDFLAGS}"\
            -o "${DEVSPACE_ROOT}/release/${NAME}" .
        shasum -a 256 "${DEVSPACE_ROOT}/release/${NAME}" > "${DEVSPACE_ROOT}/release/${NAME}".sha256
    fi
  done
done


