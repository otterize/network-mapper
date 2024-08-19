FROM golang:1.22.1 AS ebpf-buildenv

RUN apt-get update
RUN apt-get install -y clang llvm libelf-dev libbpf-dev linux-headers-generic
RUN ln -sf /usr/include/$(uname -m)-linux-gnu/asm /usr/include/asm

COPY . /src/
WORKDIR /src

RUN <<EOR
RUN go mod download
go generate -tags ebpf,linux,arm64 ./ebpf/...
go build -tags ebpf,linux,arm64 -o /src/ebpf/uprobe-counter/uprobe ./ebpf/...
chmod +x /src/ebpf/uprobe-counter/uprobe
EOR

FROM quay.io/bpfman/bpfman AS bpfman
COPY go.mod go.sum ./otterize/
COPY --from=ebpf-buildenv /src/ebpf/ /otterize/ebpf/

ENTRYPOINT ["./bpfman-rpc", "--timeout=0"]