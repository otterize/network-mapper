FROM golang:1.22.1 AS ebpf-buildenv

WORKDIR /src
COPY go.mod go.sum ./

RUN  <<EOR
apt-get update
apt-get install -y clang llvm libelf-dev libbpf-dev linux-headers-generic bpftool
ln -sf /usr/include/$(uname -m)-linux-gnu/asm /usr/include/asm
go mod download
EOR

COPY ebpf/ ./ebpf/

RUN <<EOR
go generate -tags ebpf ./ebpf/...
EOR

FROM quay.io/bpfman/bpfman AS bpfman
COPY --from=ebpf-buildenv /src/ebpf/ /otterize/ebpf/

ENTRYPOINT ["./bpfman-rpc", "--timeout=0"]

FROM --platform=$BUILDPLATFORM golang:1.22.1-alpine AS buildenv
RUN apk add --no-cache ca-certificates git protoc
RUN apk add build-base libpcap-dev
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

FROM buildenv AS test
# install dependencies for "envtest" package
RUN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20230216140739-c98506dc3b8e && \
    source <(setup-envtest use -p env) && \
    mkdir -p /usr/local/kubebuilder && \
    ln -s "$KUBEBUILDER_ASSETS" /usr/local/kubebuilder/bin
RUN go test ./node-agent/...

FROM test as builder
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /otterize-node-agent ./node-agent/cmd

# add version file
ARG VERSION
RUN echo -n $VERSION > /version

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:debug
COPY --from=builder /otterize-node-agent /otterize/bin/otterize-node-agent
COPY --from=builder /version .

ENV OTTERIZE_EBPF_SOCKET_PATH=/run/bpfman-sock/bpfman.sock
ENV OTTERIZE_EBPF_PROGRAMS_PATH=/otterize/ebpf

EXPOSE 9090
ENTRYPOINT ["/otterize/bin/otterize-node-agent"]
