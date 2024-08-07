FROM --platform=$BUILDPLATFORM golang:1.22.1 AS ebpf-buildenv

WORKDIR /src
COPY go.mod go.sum ./

RUN apt-get update
RUN apt-get install -y clang llvm libelf-dev libbpf-dev linux-headers-generic
RUN ln -sf /usr/include/$(uname -m)-linux-gnu/asm /usr/include/asm
RUN go mod download

COPY ebpf/ ./ebpf/

RUN <<EOR
go generate -tags ebpf ./ebpf/...
EOR

FROM quay.io/bpfman/bpfman AS bpfman
COPY --from=ebpf-buildenv /src/ebpf/ /otterize/ebpf/

ENTRYPOINT ["./bpfman-rpc", "--timeout=0"]