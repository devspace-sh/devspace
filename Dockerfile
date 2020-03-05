FROM alpine:3.11.3 as alpine

ARG RELEASE_VERSION=latest

RUN apk add curl tar
RUN curl -s -L "https://github.com/devspace-cloud/devspace/releases/$RELEASE_VERSION" | sed -nE 's!.*"([^"]*devspace-linux-amd64)".*!https://github.com\1!p' | xargs -n 1 curl -L -o /bin/devspace \
 && chmod +x /bin/devspace

ENTRYPOINT ["/bin/devspace"]
