//go:build ebpf

package ebpf

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target $GOARCH -cc clang -no-strip -cflags "-O2 -g -Wall" Bpf ./agent.ebpf.c -- -I.:/usr/include/bpf:/usr/include/linux -I./include -I./gotls -I./openssl -DTARGET_ARCH_$GOARCH
