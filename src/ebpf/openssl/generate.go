//go:build ebpf

package openssl

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target $GOARCH -cc clang -no-strip -cflags "-O2 -g -Wall" bpf ./counter.c -- -I.:/usr/include/bpf:/usr/include/linux
