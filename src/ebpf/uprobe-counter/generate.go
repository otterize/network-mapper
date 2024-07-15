//go:build ebpf

package uprobe_counter

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64 -cc clang -no-strip -cflags "-O2 -g -Wall" bpf ./counter.c -- -I.:/usr/include/bpf:/usr/include/linux
