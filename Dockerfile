FROM alpine:3 as alpine

ARG RELEASE_VERSION=latest

RUN apk add --update-cache curl tar docker git

RUN curl -L -o /bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.17.3/bin/linux/amd64/kubectl \
 && chmod +x /bin/kubectl

RUN curl -s -L "https://github.com/loft-sh/devspace/releases/download/$RELEASE_VERSION/devspace-linux-amd64" -o /bin/devspace \
 && chmod +x /bin/devspace
