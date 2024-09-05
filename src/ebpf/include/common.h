#pragma once

#include "filters.h"

static __inline __u32 get_pid() {
    return bpf_get_current_pid_tgid() >> 32;
}

static __inline int should_trace() {
    // gets the current (real) PID
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // gets the current 'task' which is a linux kernel struct that represents a process
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();

    // a bit of magic, but this gets the inode of the current PID namespace
    // equivalent to running `readlink /proc/self/ns/pid`
    int nsInode = BPF_CORE_READ(task, group_leader, nsproxy, pid_ns_for_children, ns.inum);

    bpf_printk("PID namespace: %llu", nsInode);

    struct target_t *pTarget = bpf_map_lookup_elem(&targets, &nsInode);

    if (pTarget == 0) {
        bpf_printk("tracing disabled for pid, pid: %llu", pid);
        return 0;
    }

    return pTarget->enabled;
}

static __inline void send_event(struct pt_regs *ctx, __u64 buf, __u64 size, __u64 total_size, enum direction_t direction) {
    bpf_printk("reading: %x %d", buf, size);

    long err;

    // Initialize the event from the map.
    struct ssl_event_t *event = bpf_map_lookup_elem(&ssl_event, &ZERO);
    if (event == NULL){
        bpf_printk("error creating ssl_event_t");
        return;
    }

    event->meta.pid = get_pid();
    event->meta.timestamp = bpf_ktime_get_ns();
    event->meta.data_size = size;
    event->meta.direction = direction;
    event->meta.total_size = total_size;

    // Read the data from the buffer
    err = bpf_probe_read(event->data, size, (void *)buf);
    if (err) {
        bpf_printk("error reading data");
        return;
    }

    // Check if we should send the event
    if(!should_send_event(event)) return;

    // Send the event to the event array
    err = bpf_perf_event_output(
        ctx,
        &ssl_events,
        BPF_F_CURRENT_CPU,
        event,
        (sizeof(struct ssl_event_meta_t) + event->meta.data_size) & (MAX_DATA_SIZE - 1)
    );

    if (err != 0) {
        bpf_printk("error sending event: %d", err);
        return;
    }
}