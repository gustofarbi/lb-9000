FROM golang:1.21.0-alpine3.18

WORKDIR /go/src/app

COPY main.go go.mod ./

RUN go build -o /go/bin/app

CMD ["/go/bin/app"]
