FROM golang:1.17-alpine

ADD . /go/src/app
WORKDIR /go/src/app

RUN cd /go/src/app && go build -o app . && chmod +x /go/src/app/app

CMD ["/go/src/app/app"]
