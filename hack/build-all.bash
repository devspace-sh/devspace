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
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
DATE=$(date "+%Y-%m-%d")
BUILD_PLATFORM=$(uname -a | awk '{print tolower($1);}')

echo "Current working directory is $(pwd)"
echo "PATH is $PATH"
echo "GOPATH is $GOPATH"

if [[ "$(pwd)" != "${DEVSPACE_ROOT}" ]]; then
  echo "you are not in the root of the repo" 1>&2
  echo "please cd to ${DEVSPACE_ROOT} before running this script" 1>&2
  exit 1
fi

GO_BUILD_CMD="go build -a"
GO_BUILD_LDFLAGS="-s -w -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${DATE} -X main.version=${VERSION} -X github.com/loft-sh/devspace/pkg/devspace/config/localcache.EncryptionKey=$ENCRYPTION_KEY"

if [[ -z "${DEVSPACE_BUILD_PLATFORMS}" ]]; then
    DEVSPACE_BUILD_PLATFORMS="linux windows darwin"
fi

if [[ -z "${DEVSPACE_BUILD_ARCHS}" ]]; then
    DEVSPACE_BUILD_ARCHS="amd64 386 arm64"
fi

# Create the release directory
mkdir -p "${DEVSPACE_ROOT}/release"

# Install Helm 4
echo "Installing helm"
HELM_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
HELM_ARCH=$(uname -m)
case "${HELM_ARCH}" in
  x86_64)
    HELM_ARCH="amd64"
    ;;
  aarch64|arm64)
    HELM_ARCH="arm64"
    ;;
  i386|i686)
    HELM_ARCH="386"
    ;;
esac
HELM_PLATFORM="${HELM_OS}-${HELM_ARCH}"
HELM_VERSION=$(sed -nE 's/^const helmVersion = "([^"]+)"/\1/p' pkg/util/downloader/commands/helm_v4.go)
if [[ -z "${HELM_VERSION}" ]]; then
  echo "unable to determine Helm version" 1>&2
  exit 1
fi
curl -s "https://get.helm.sh/helm-${HELM_VERSION}-${HELM_PLATFORM}.tar.gz" > helm4.tar.gz && tar -zxvf helm4.tar.gz "${HELM_PLATFORM}/helm" && chmod +x "${HELM_PLATFORM}/helm"

# Pull the component chart
COMPONENT_CHART_VERSION=$(cat pkg/devspace/deploy/deployer/helm/client.go | grep 'Version: "' | sed -nE 's/[^"]+"(.+)",\s*/\1/p')
"${HELM_PLATFORM}/helm" pull component-chart --repo https://charts.devspace.sh --version $COMPONENT_CHART_VERSION

# Move ui.tar.gz to releases
echo "Moving ui"
if [[ -f "${DEVSPACE_ROOT}/ui.tar.gz" ]]; then
  mv "${DEVSPACE_ROOT}/ui.tar.gz" "${DEVSPACE_ROOT}/release/ui.tar.gz"
elif [[ ! -f "${DEVSPACE_ROOT}/release/ui.tar.gz" ]]; then
  echo "ui tarball not found; run ./hack/build-ui.bash first" 1>&2
  exit 1
fi
shasum -a 256 "${DEVSPACE_ROOT}/release/ui.tar.gz" > "${DEVSPACE_ROOT}/release/ui.tar.gz".sha256

# build devspace helper
echo "Building devspace helper"
GOARCH=amd64 GOOS=linux go build -ldflags "-s -w -X github.com/loft-sh/devspace/helper/cmd.version=${VERSION}" -o "${DEVSPACE_ROOT}/release/devspacehelper" helper/main.go
# upx "${DEVSPACE_ROOT}/release/devspacehelper" #compress devspacehelper
shasum -a 256 "${DEVSPACE_ROOT}/release/devspacehelper" > "${DEVSPACE_ROOT}/release/devspacehelper".sha256

GOARCH=arm64 GOOS=linux go build -ldflags "-s -w -X github.com/loft-sh/devspace/helper/cmd.version=${VERSION}" -o "${DEVSPACE_ROOT}/release/devspacehelper-arm64" helper/main.go
# FIXME: this is not working for any number of arguments/flags
# upx "${DEVSPACE_ROOT}/release/devspacehelper-arm64" #compress devspacehelper
shasum -a 256 "${DEVSPACE_ROOT}/release/devspacehelper-arm64" > "${DEVSPACE_ROOT}/release/devspacehelper-arm64".sha256

# build bin data
GO_BINDATA="$(command -v go-bindata || true)"
if [[ -z "${GO_BINDATA}" ]]; then
  GO_BINDATA="$(go env GOPATH)/bin/go-bindata"
fi
if [[ ! -x "${GO_BINDATA}" ]]; then
  echo "go-bindata not found; install it with: go install github.com/go-bindata/go-bindata/go-bindata@latest" 1>&2
  exit 1
fi
"${GO_BINDATA}" -o assets/assets.go -pkg assets release/devspacehelper release/ui.tar.gz component-chart-$COMPONENT_CHART_VERSION.tgz

for OS in ${DEVSPACE_BUILD_PLATFORMS[@]}; do
  for ARCH in ${DEVSPACE_BUILD_ARCHS[@]}; do
    NAME="devspace-${OS}-${ARCH}"
    if [[ "${OS}" == "windows" ]]; then
      NAME="${NAME}.exe"
    fi

    # darwin 386 is deprecated and shouldn't be used anymore
    if [[ "${ARCH}" == "386" && "${OS}" == "darwin" ]]; then
        echo "Building for ${OS}/${ARCH} not supported."
        continue
    fi

    # arm64 build is only supported for darwin
    if [[ "${ARCH}" == "arm64" && "${OS}" == "windows" ]]; then
        echo "Building for ${OS}/${ARCH} not supported."
        continue
    fi

    echo "Building for ${OS}/${ARCH}"

    # build darwin with CGO_ENABLED=1
    if [[ "${OS}" == "darwin" ]]; then
      CGO_ENABLED=1
    else
      CGO_ENABLED=0
    fi

    # build the DevSpace binary
    CGO_ENABLED=${CGO_ENABLED} GOARCH=${ARCH} GOOS=${OS} ${GO_BUILD_CMD} -ldflags "${GO_BUILD_LDFLAGS}"\
                  -o "${DEVSPACE_ROOT}/release/${NAME}" .
    shasum -a 256 "${DEVSPACE_ROOT}/release/${NAME}" > "${DEVSPACE_ROOT}/release/${NAME}".sha256
  done
done
