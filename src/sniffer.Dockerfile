FROM golang:1.18-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go generate ./sniffer/...

FROM buildenv as test
RUN go test ./sniffer/...

FROM test as builder
RUN go build -o /main ./sniffer/cmd

FROM alpine as release
RUN apk add --no-cache ca-certificates libpcap
WORKDIR /
COPY --from=builder /main /main
RUN chmod +x /main

ENTRYPOINT ["/main"]
