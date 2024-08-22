//go:build ignore
#include "vmlinux_aarch64.h"
#include "vmlinux_x86_64.h"
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

const __u32 MAX_SIZE = 512;
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
} targets SEC(".maps");

int shouldTrace() {
    // gets the current (real) PID
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // gets the current 'task' which is a linux kernel struct that represents a process
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();

    // a bit of magic, but this gets the inode of the current PID namespace
    // equivalent to running `readlink /proc/self/ns/pid`
    int nsInode = BPF_CORE_READ(task, group_leader, nsproxy, pid_ns_for_children, ns.inum);

    bpf_printk("PID namespace: %llu", nsInode);

    _Bool *pTarget = bpf_map_lookup_elem(&targets, &nsInode);

    if (pTarget == 0) {
        bpf_printk("tracing disabled for pid, pid: %llu", pid);
        return 0;
    }

    return *pTarget;
}

__u32 get_pid() {
    return bpf_get_current_pid_tgid() >> 32;
}

SEC("uprobe/otterize_SSL_write")
void BPF_KPROBE(otterize_SSL_write, void* ssl, uintptr_t buffer, int num) {
    if (!shouldTrace()) {
        return;
    }

    bpf_printk("entering SSL_write");

    // capture the cleartext buffer and size
    struct ssl_context_t context = {
        .buffer = buffer,
        .size = num
    };

    __u64 key = bpf_get_current_pid_tgid();
    long err = bpf_map_update_elem(&ssl_contexts, &key, &context, BPF_ANY);

    if (err != 0) {
        bpf_printk("capturing SSL_write input: update_elem failed");
        return;
    }
}

SEC("uretprobe/otterize_SSL_write_ret")
void BPF_KRETPROBE(otterize_SSL_write_ret) {
    if (!shouldTrace()) {
        return;
    }

    bpf_printk("entering SSL_write_ret");

    __u64 key = bpf_get_current_pid_tgid();
    void* pContext = bpf_map_lookup_elem(&ssl_contexts, &key);

    if (pContext == NULL) {
        bpf_printk("pContext is null");
        return;
    }

    struct ssl_context_t context = {
        .buffer = 0,
        .size = 0
    };

    long err = bpf_probe_read(&context, sizeof(struct ssl_context_t), pContext);

    if (err != 0) {
        bpf_printk("bpf_probe_read failed");
        return;
    }

    int zero = 0;
    struct ssl_event_t *event = bpf_map_lookup_elem(&ssl_event, &zero);

    if (event == NULL) {
        bpf_printk("failed to create ssl_event_t");
        return;
    }

    event->pid = get_pid();
    event->size = context.size;

    if (event->size <= 0) {
        // not supposed to happen, but the verifier can't know that
        return;
    } else if (event->size > MAX_SIZE) {
        event->size = MAX_SIZE;
    }

    // copy the cleartext buffer to the event struct
    // the verifier doesn't let us use event->size, or any other variable, as it
    // can't ensure that it's within bounds. So we use a constant the size of the
    // output buffer. This appeases the verifier, but I'm not sure what is the effect of
    // reading beyond (context.buffer + context.size). ???
    err = bpf_probe_read_user(&event->data, MAX_SIZE, (char*)context.buffer);

    if (err != 0) {
        bpf_printk("bpf_probe_read_user failed");
        return;
    }

    err = bpf_perf_event_output(ctx, &ssl_events, BPF_F_CURRENT_CPU, event, sizeof(struct ssl_event_t));

    if (err != 0) {
        bpf_printk("bpf_perf_event_output failed: %d", err);
        return;
    }

    // delete the context
    bpf_map_delete_elem(&ssl_contexts, &key);

    return;
}

char _license[] SEC("license") = "GPL";