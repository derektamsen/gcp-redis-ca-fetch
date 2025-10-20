FROM golang:1.25.3-alpine3.22 AS build

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/gcp-fetch-redis-certs ./...

FROM alpine:3.22

RUN adduser -s /sbin/nologin -DH -u 1000 app

COPY --from=build /usr/local/bin/gcp-fetch-redis-certs /usr/local/bin/gcp-fetch-redis-certs

USER app

ENTRYPOINT ["/usr/local/bin/gcp-fetch-redis-certs"]
