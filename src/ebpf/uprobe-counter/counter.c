//go:build ignore
#include <linux/types.h>
#include <linux/bpf.h>
#include <bpf/bpf_tracing.h>
#include <asm/ptrace.h>

const __u32 MAX_SIZE = 1024;

struct ssl_event_t {
    __u32 pid;
//    __u64 timestamp;
    __u32 size;
    char data[MAX_SIZE];
};

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct ssl_event_t);
} ssl_data SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u64);
    __type(value, void*);
    __uint(max_entries, 1024);
} ssl_write SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u32));
} events SEC(".maps");

SEC("uprobe/openssl_SSL_write_entry")
static __u32 openssl_SSL_write_entry(struct pt_regs *ctx) {
    __u64 pid = bpf_get_current_pid_tgid();

    const void* buf = (const void*)PT_REGS_PARM1(ctx);
    bpf_map_update_elem(&ssl_write, &pid, &buf, 0);

    return 0;
}

SEC("uprobe/openssl_SSL_write_exit")
static __u32 openssl_SSL_write_exit(struct pt_regs *ctx) {
    __u64 pid = bpf_get_current_pid_tgid();

    void **buf = bpf_map_lookup_elem(&ssl_write, &pid);

    if (!buf) {
        return 0;
    }

    const __s32 ret = PT_REGS_RC(ctx);

    int zero = 0;
    struct ssl_event_t* event = bpf_map_lookup_elem(&ssl_data, &zero);

    if (!event) {
        return 0;
    }

    event->pid = pid;
    event->size = ret > MAX_SIZE ? MAX_SIZE : ret;
    bpf_probe_read_user(&event->data, event->size, *buf);

    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &event, sizeof(event));

    bpf_map_delete_elem(&ssl_write, &pid);

    return 0;
}
