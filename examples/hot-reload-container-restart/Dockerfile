FROM golang:1.27rc1-alpine

ADD . /go/src/app
WORKDIR /go/src/app

RUN cd /go/src/app

ENTRYPOINT ["go", "run", "main.go"]
