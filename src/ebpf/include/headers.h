#ifdef TARGET_ARCH_amd64
#include "vmlinux_x86_64.h"
#endif

#ifdef TARGET_ARCH_arm64
#include "vmlinux_aarch64.h"
#endif

#include <bpf/bpf_tracing.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
