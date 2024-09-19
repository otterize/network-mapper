//go:build ebpf

package ebpf

//go:generate sh -c "bpftool btf dump file /sys/kernel/btf/vmlinux format c > ./include/vmlinux.h"
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target $GOARCH -cc clang -no-strip -cflags "-O2 -g -Wall" Bpf ./agent.ebpf.c -- -I.:/usr/include/bpf:/usr/include/linux -I./ -DTARGET_ARCH_$GOARCH