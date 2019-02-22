FROM golang:1.11

RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
ADD . $GOPATH/src/app

RUN cd $GOPATH/src/app && go get ./... && go build . && cd /app && chmod +x $GOPATH/src/app/app

WORKDIR /app
CMD ["/go/src/app/app"]
