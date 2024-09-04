//go:build ignore

#include "headers.h"
#include "maps.h"
#include "common.h"


SEC("uprobe/otterize_SSL_write")
void BPF_KPROBE(otterize_SSL_write, void* ssl, uintptr_t buffer, int num) {
    if (!shouldTrace()) return;

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
    if (!shouldTrace()) return;

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

    // Calculate the size to read
    __u64 size_to_read = context.size;
    if (size_to_read > MAX_CHUNK_SIZE) size_to_read = MAX_CHUNK_SIZE;

    // Send the event
    send_event(ctx, context.buffer, size_to_read, context.size, EGRESS);

    // delete the context
    bpf_map_delete_elem(&ssl_contexts, &key);

    return;
}
