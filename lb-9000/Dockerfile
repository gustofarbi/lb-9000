FROM golang:1.22.3-alpine3.19

WORKDIR /go/src/app

COPY . ./

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /go/bin/app

#ENTRYPOINT ["tail", "-f", "/dev/null"]
CMD ["/go/bin/app"]
