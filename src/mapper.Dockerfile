FROM golang:1.18-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go generate ./mapper/...

FROM buildenv as test
RUN --mount=type=cache,target=/root/.cache/go-build go test ./mapper/...

FROM buildenv as builder
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /main ./mapper/cmd

FROM alpine as release
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /main /main
RUN chmod +x /main

EXPOSE 9090
ENTRYPOINT ["/main"]
