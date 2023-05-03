FROM mcr.microsoft.com/devcontainers/go:0-1.19-bullseye

ARG TARGETOS
ARG TARGETARCH

RUN curl -L -o kind "https://kind.sigs.k8s.io/dl/v0.17.0/kind-linux-${TARGETARCH}" && install -c -m 0755 kind /usr/local/bin
