//go:build ignore
#include "vmlinux_aarch64.h"
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

const __u32 MAX_SIZE = 1024;
const __u32 MAX_ENTRIES_HASH = 4096;

struct ssl_event_t {
    __u32 pid;
    __u64 timestamp;
    __u32 size;
    __u8 data[MAX_SIZE];
};

struct ssl_context_t {
    __u64 size;
    __u64 buffer;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, __u64);
    __type(value, struct ssl_context_t);
//    __uint(pinning, LIBBPF_PIN_BY_NAME);
} ssl_contexts SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __type(key, __u32);
    __type(value, struct ssl_event_t);
    __uint(max_entries, 1);
} ssl_event SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} ssl_events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_HASH);
    __type(key, __u32);
    __type(value, _Bool);
} pid_targets SEC(".maps");

int should_trace() {
    __u64 pid = bpf_get_current_pid_tgid();

    struct task_struct *task = (struct task_struct *)bpf_get_current_task();

    int nsInode = BPF_CORE_READ(task, group_leader, nsproxy, pid_ns_for_children, ns.inum);

    {
        char msg[] = "pid ns inode: %llu";
        bpf_trace_printk(msg, sizeof(msg), nsInode);
    }

    _Bool *pTarget = bpf_map_lookup_elem(&pid_targets, &nsInode);

    if (pTarget == 0) {
        {
            char msg[] = "tracing disabled for pid, pid: %llu";
            bpf_trace_printk(msg, sizeof(msg), pid);
        }
        return 0;
    }

    return *pTarget;
}

SEC("uprobe/otterize_SSL_write")
void BPF_KPROBE(otterize_SSL_write, void* ssl, uintptr_t buffer, int num) {
    if (!should_trace()) {
        return;
    }

    __u64 pid = bpf_get_current_pid_tgid();

    {
        char msg[] = "openssl_SSL_write_entry, buffer: %p, size: %d, pid: %llu";
        bpf_trace_printk(msg, sizeof(msg), buffer, num, pid);
    }

    struct ssl_context_t context = {
        .buffer = buffer,
        .size = num
    };

    long err = bpf_map_update_elem(&ssl_contexts, &pid, &context, BPF_ANY);

    if (err != 0) {
        char msg[] = "bpf_map_update_elem failed";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }
}

SEC("uretprobe/otterize_SSL_write_ret")
void BPF_KRETPROBE(otterize_SSL_write_ret) {
    __u64 pid = bpf_get_current_pid_tgid();

    {
        char msg[] = "openssl_SSL_write_exit, pid: %llu";
        bpf_trace_printk(msg, sizeof(msg), pid);
    }

    void *pContext = bpf_map_lookup_elem(&ssl_contexts, &pid);

    if (pContext == 0) {
        char msg[] = "pContext is null";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }


//    int result = PT_REGS_RC(ctx);
//
//
    struct ssl_context_t context = {
        .buffer = 0,
        .size = 0
    };
    long err = bpf_probe_read(&context, sizeof(struct ssl_context_t), pContext);

    if (err != 0) {
        char msg[] = "bpf_probe_read failed";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }

    {
        char msg[] = "exit, context: %p, context.size: %llu";
        bpf_trace_printk(msg, sizeof(msg), context.buffer, context.size);
    }

    int zero = 0;
    struct ssl_event_t *event = bpf_map_lookup_elem(&ssl_event, &zero);

    if (event == 0) {
        char msg[] = "event is null";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }

    event->pid = pid;
    event->size = context.size;

    {
        char msg[] = "pid: %d, size: %d";
        bpf_trace_printk(msg, sizeof(msg), event->pid, event->size);
    }

    if (event->size <= 0) {
        return;
    }

    if (event->size > MAX_SIZE) {
        event->size = MAX_SIZE;
    }

    err = bpf_probe_read_user(&event->data, event->size, (char*)context.buffer);

    if (err != 0) {
        char msg[] = "bpf_probe_read_user failed";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }

    err = bpf_perf_event_output(ctx, &ssl_events, BPF_F_CURRENT_CPU, event, sizeof(struct ssl_event_t));

    if (err != 0) {
        char msg[] = "bpf_perf_event_output failed";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }

    {
        char msg[] = "bpf_perf_event_output success";
        bpf_trace_printk(msg, sizeof(msg));
        return;
    }

//    bpf_map_delete_elem(&ssl_buffers, &pid);

    return;
}

//static struct ssl_context_t lookup_buffer(struct pt_regs* ctx, void* map, __u64 key) {
//    struct ssl_context_t* pContext = bpf_map_lookup_elem(map, &key);
//    struct ssl_context_t buffer = {
//            .buffer = 0,
//            .size = 0
//    };
//
//    if (pContext != NULL) {
//        bpf_probe_read(&buffer, sizeof(struct ssl_context_t), pContext);
//    }
//
//    return buffer;
//}

char _license[] SEC("license") = "GPL";