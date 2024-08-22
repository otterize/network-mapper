FROM golang:1.22.1 AS ebpf-buildenv

RUN apt-get update
RUN apt-get install -y clang libelf-dev libbpf-dev linux-headers-generic
RUN ln -sf /usr/include/$(uname -m)-linux-gnu/asm /usr/include/asm

COPY . /src/
WORKDIR /src

RUN --mount=type=cache,target="/root/.cache/go-build" <<EOR
set -ex
go mod download
go generate -tags ebpf ./ebpf/...
EOR

FROM --platform=$BUILDPLATFORM golang:1.22.1-alpine AS buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM --platform=$BUILDPLATFORM buildenv AS test
# install dependencies for "envtest" package
#
#RUN go test ./node-agent/...

FROM --platform=$BUILDPLATFORM test AS builder
ARG TARGETOS
ARG TARGETARCH

COPY --from=ebpf-buildenv /src/ebpf /src/ebpf
RUN --mount=type=cache,target="/root/.cache/go-build" <<EOR
set -ex
CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /otterize-node-agent ./node-agent/cmd/agent
EOR

# add version file
ARG VERSION
RUN echo -n $VERSION > /version

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM ubuntu:24.04
COPY --from=builder /otterize-node-agent /otterize/bin/otterize-node-agent
COPY --from=builder /version .

EXPOSE 9090
ENTRYPOINT ["/otterize/bin/otterize-node-agent"]
