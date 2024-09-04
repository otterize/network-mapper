#pragma once

static __inline bool shouldSend() {
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