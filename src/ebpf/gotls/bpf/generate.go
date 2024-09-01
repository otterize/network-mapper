//go:build ebpf

package bpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target $GOARCH -cc clang -no-strip -cflags "-O2 -g -Wall" Bpf ./gotls.bpf.c -- -I.:/usr/include/bpf:/usr/include/linux
