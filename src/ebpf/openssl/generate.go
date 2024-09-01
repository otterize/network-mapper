//go:build ebpf

package openssl

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target $GOARCH -cc clang -no-strip -cflags "-O2 -g -Wall" openssl ./openssl.ebpf.c -- -I.:/usr/include/bpf:/usr/include/linux -I/src/ebpf/include -DTARGET_ARCH_$GOARCH
