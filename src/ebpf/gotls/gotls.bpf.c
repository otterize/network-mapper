//go:build ignore
#include <linux/ptrace.h>
#include <linux/types.h>
#include <linux/bpf.h>
#include <bpf/bpf_tracing.h>

#define MAX_MSG_SIZE 30720
#define CHUNK_LIMIT 4

// ####################################################################### //
// Structs
// ####################################################################### //

struct go_slice {
    void* ptr;
    int len;
    int cap;
};

struct go_fn_info {
    __u64 base_addr;
    __u64 stack_addr;
    __u64 r0_stack_addr;
    __u64 r1_stack_addr;
    __u64 r2_stack_addr;
    __u64 r3_stack_addr;
};

struct event {
    struct meta_t {
    	__u32 pid;
    	__u64 pos;
    	__u64 buf_size;
    	__u64 msg_size;
    } meta;
	char msg[MAX_MSG_SIZE];
};

// ####################################################################### //
// BPF Maps
// ####################################################################### //

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct go_fn_info);
} go_fn_info_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct event);
} events_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} events SEC(".maps");

// Force emitting struct event into the ELF.
const struct event *unused __attribute__((unused));

// ####################################################################### //
// Functions
// ####################################################################### //

// Read and submit the buffer data - split into chunks if necessary.
static __inline void read_buffer(struct pt_regs *ctx, struct go_slice buf, struct event *event) {
    __u64 bytes_sent = 0;
    unsigned int i;

    // Get the TGID (process ID) for the event
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 tgid = pid_tgid >> 32; // The upper 32 bits contain the process ID

    #pragma unroll
    for (i = 0; i < CHUNK_LIMIT; ++i) {
        const __u64 bytes_remaining = buf.len - bytes_sent;

        // Calculate the size to read
        __u64 size_to_read = buf.len;
        if (size_to_read > MAX_MSG_SIZE) size_to_read = MAX_MSG_SIZE;

        event->meta.pid = tgid;
        event->meta.pos = bytes_sent;
        event->meta.buf_size = buf.len;
        event->meta.msg_size = size_to_read;

        bpf_probe_read(event->msg, size_to_read, buf.ptr);

        // Submit the event to the event array
        bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, event, sizeof(*event));

        if (size_to_read == bytes_remaining) break;
        bytes_sent += size_to_read;
    }
}

SEC("uprobe/go_tls_write")
int gotls_write_hook(struct pt_regs *ctx) {
    // Function signature: func (c *Conn) Write(b []byte) (int, error)
    // Function symbol:    crypto/tls.(*Conn).Write

    bpf_printk("uprobe invoked on crypto/tls connection write \n");

    int key = 0;  // Since we have only one entry

    // Initialize the event from the map.
    struct event *event = bpf_map_lookup_elem(&events_map, &key);
    if (!event) return 0;

    // Get the function information.
    struct go_fn_info *fn_info;
    fn_info = bpf_map_lookup_elem(&go_fn_info_map, &key);
    if (!fn_info) return 0;

    // Get the stack pointer.
    const void* sp = (const void*)PT_REGS_SP(ctx);

    // Get the buffer pointer - should be equal to (const char*)PT_REGS_PARM2(ctx)
    char* buf_ptr;
    bpf_probe_read(&buf_ptr, sizeof(char*), sp + fn_info->r1_stack_addr);

    // Get the buffer slice - pointer should be equal to (const char*)PT_REGS_PARM2(ctx)
    struct go_slice buf;
    bpf_probe_read(&buf, sizeof(struct go_slice), sp + fn_info->r1_stack_addr);
    bpf_printk("slice: %x %d %d", buf.ptr, buf.len, buf.cap);

    // Read the buffer data.
    read_buffer(ctx, buf, event);

    return 0;
}

char __license[] SEC("license") = "Dual MIT/GPL";
