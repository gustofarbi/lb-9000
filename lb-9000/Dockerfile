FROM golang:1.23.4-alpine3.21

WORKDIR /go/src/app

COPY go.mod go.sum ./
COPY lb-9000 ./lb-9000

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /go/bin/app lb-9000/main.go

#ENTRYPOINT ["tail", "-f", "/dev/null"]
CMD ["/go/bin/app"]
