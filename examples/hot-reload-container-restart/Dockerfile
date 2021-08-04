FROM golang:1.16.4-alpine

ADD . /go/src/app
WORKDIR /go/src/app

RUN cd /go/src/app && go get ./...

ENTRYPOINT ["go", "run", "main.go"]
