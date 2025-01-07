FROM golang:1.23.4-alpine3.21

WORKDIR /go/src/app

COPY internal/client/main.go go.mod go.sum ./

RUN go build -o /go/bin/app

CMD ["/go/bin/app"]
