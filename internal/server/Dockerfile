FROM golang:1.22.3-alpine3.19

WORKDIR /go/src/app

COPY internal/server/main.go go.mod ./

RUN go build -o /go/bin/app

CMD ["/go/bin/app"]
