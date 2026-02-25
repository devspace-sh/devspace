FROM golang:1.26rc2-alpine

ADD . /go/src/app
WORKDIR /go/src/app

RUN cd /go/src/app

ENTRYPOINT ["go", "run", "main.go"]
