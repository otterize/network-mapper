//go:build ignore

// Common header for all eBPF programs
#include "include/headers.h"

// Event logic
#include "include/events/events.h"
#include "include/events/events.c"

#include "include/filters/filters.h"
#include "include/filters/filters.c"

// All eBPF programs
#include "gotls/gotls.ebpf.c"
#include "openssl/openssl.ebpf.c"

char _license[] SEC("license") = "GPL";