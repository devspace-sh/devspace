FROM alpine:3 as alpine

ARG RELEASE_VERSION=latest

RUN apk add --update-cache curl tar docker

RUN curl -L -o /bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.17.3/bin/linux/amd64/kubectl \
 && chmod +x /bin/kubectl

RUN curl -s -L "https://github.com/loft-sh/devspace/releases/$RELEASE_VERSION" | sed -nE 's!.*"([^"]*devspace-linux-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o /bin/devspace \
 && chmod +x /bin/devspace
