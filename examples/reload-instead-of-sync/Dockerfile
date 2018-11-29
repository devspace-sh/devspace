FROM golang:1.11

RUN mkdir -p "$GOPATH/src/app" && ln -s $GOPATH/src/app /app
ADD . $GOPATH/src/app

RUN cd $GOPATH/src/app && cd /app

WORKDIR /app
CMD ["go", "run", "main.go"]
