//go:build ignore

// Common header for all eBPF programs
#include "headers.h"
#include "maps.h"
#include "filters.h"
#include "common.h"

// All eBPF programs
#include "gotls.ebpf.c"
#include "openssl.ebpf.c"

char _license[] SEC("license") = "GPL";