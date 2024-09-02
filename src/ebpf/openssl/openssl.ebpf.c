//go:build ignore

#include "headers.h"
#include "maps.h"
#include "common.h"


SEC("uprobe/otterize_SSL_write")
void BPF_KPROBE(otterize_SSL_write, void* ssl, uintptr_t buffer, int num) {
    if (!shouldTrace()) {
        return;
    }

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

    struct ssl_event_t *event = bpf_map_lookup_elem(&ssl_event, &ZERO);

    if (event == NULL) {
        bpf_printk("failed to create ssl_event_t");
        return;
    }

    event->meta.pid = get_pid();
    event->meta.data_size = context.size;

    if (context.size <= 0) {
        // not supposed to happen, but the verifier can't know that
        return;
    } else if (context.size > MAX_CHUNK_SIZE) {
        event->meta.data_size = MAX_CHUNK_SIZE;
    }

    // copy the cleartext buffer to the event struct
    // the verifier doesn't let us use event->size, or any other variable, as it
    // can't ensure that it's within bounds. So we use a constant the size of the
    // output buffer. This appeases the verifier, but I'm not sure what is the effect of
    // reading beyond (context.buffer + context.size). ???
    err = bpf_probe_read_user(&event->data, context.size & (MAX_CHUNK_SIZE - 1), (char*)context.buffer);

    if (err != 0) {
        bpf_printk("bpf_probe_read_user failed");
        return;
    }

    err = bpf_perf_event_output(
        ctx,
        &ssl_events,
        BPF_F_CURRENT_CPU,
        event,
        (sizeof(struct ssl_event_meta_t) + event->meta.data_size) & (MAX_CHUNK_SIZE - 1)
    );

    if (err != 0) {
        bpf_printk("bpf_perf_event_output failed: %d", err);
        return;
    }

    // delete the context
    bpf_map_delete_elem(&ssl_contexts, &key);

    return;
}

char _license[] SEC("license") = "GPL";