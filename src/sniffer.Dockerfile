FROM --platform=linux/amd64 golang:1.18-alpine as buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
ARG GITHUB_TOKEN
RUN --mount=type=secret,id=github_token \
    if [ -f /run/secrets/github_token ]; then export GITHUB_TOKEN=$(cat /run/secrets/github_token); fi && \
    git config --global url."https://$GITHUB_TOKEN@github.com/".insteadOf "https://github.com/" &&  \
    go mod download &&  \
    git config --global --unset url."https://$GITHUB_TOKEN@github.com/".insteadOf

COPY . .
RUN go generate ./sniffer/...

FROM buildenv as test
RUN go test ./sniffer/... && echo dep > /dep

# We start from the base image again, only this time it's using the target arch instead of always amd64. This is done to make the build faster.
# Unlike the mapper, it can't be amd64 throughout and use Go's cross-compilation, since the sniffer depends on libpcap (C library).
FROM golang:1.18-alpine as builder
COPY --from=test /dep /dep
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY . .
ARG GITHUB_TOKEN
RUN --mount=type=secret,id=github_token \
    if [ -f /run/secrets/github_token ]; then export GITHUB_TOKEN=$(cat /run/secrets/github_token); fi && \
    git config --global url."https://$GITHUB_TOKEN@github.com/".insteadOf "https://github.com/" &&  \
    go mod download &&  \
    git config --global --unset url."https://$GITHUB_TOKEN@github.com/".insteadOf
RUN go build -o /main ./sniffer/cmd

FROM alpine as release
RUN apk add --no-cache ca-certificates libpcap
WORKDIR /
COPY --from=builder /main /main
RUN chmod +x /main

ENTRYPOINT ["/main"]
