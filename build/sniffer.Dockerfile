FROM golang:1.22.1-alpine AS buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM buildenv AS test
RUN go test ./sniffer/... && echo dep > /dep

# We start from the base image again, only this time it's using the target arch instead of always amd64. This is done to make the build faster.
# Unlike the mapper, it can't be amd64 throughout and use Go's cross-compilation, since the sniffer depends on libpcap (C library).
FROM golang:1.22.1-alpine AS builder
COPY --from=test /dep /dep
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY . .
RUN go mod download
RUN go build -trimpath -o /main ./sniffer/cmd

# add version file
ARG VERSION
RUN echo -n $VERSION > /version

FROM alpine AS release
RUN apk add --no-cache ca-certificates libpcap
WORKDIR /
COPY --from=builder /main /main
RUN chmod +x /main
COPY --from=builder /version .

ENTRYPOINT ["/main"]
