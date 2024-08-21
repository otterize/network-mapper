//go:build ignore
#include <linux/ptrace.h>
#include <linux/types.h>
#include <linux/bpf.h>
#include <bpf/bpf_tracing.h>

// The maximum percpu map element size is 32KB - we set it at 30KB.
// Source: https://github.com/iovisor/bcc/issues/2519
#define MAX_PER_CPU_EL_SIZE 30720 // 30KB max buffer size

// Set max chunks we allow the buffer to be split into.
// TODO: currently this is a made up number
#define MAX_CHUNKS 4

// TODO: add some return values and error handling

// --------------------------------------------------------------------
// Gets the ID of the go routine - we are accessing the go id depending on the architecture.
// This code is supported only for ARM64 and x86_64 architectures and go versions 1.18 and above.
// For ARM64 - https://go.googlesource.com/go/+/refs/heads/master/src/cmd/compile/abi-internal.md#arm64-architecture
// For x86_64 - https://go.googlesource.com/go/+/refs/heads/master/src/cmd/compile/abi-internal.md#amd64-architecture
#define GOROUTINE_ID_OFFSET 152

#if defined(bpf_target_x86)
#define GO_G_STRUCT_PTR(x) ((x)->r14) // For x86_64 the g struct is stored in register number 14

#elif defined(bpf_target_arm64)
#define PT_REGS_ARM64 const volatile struct user_pt_regs
#define GO_G_STRUCT_PTR(x) (((PT_REGS_ARM64 *)(x))->regs[28]) // For ARM64 the g struct is stored in register number 28

#endif

// TODO: check if amd64 stores the slice in different registers - looks like its on [&ctx->bx, &ctx->cx, &ctx->di]
#define GO_TLS_BUFFER_PTR(ctx) ((__u64)PT_REGS_PARM2(ctx))
#define GO_TLS_BUFFER_LEN(ctx) ((int)PT_REGS_PARM3(ctx))
#define GO_TLS_BUFFER_CAP(ctx) ((int)PT_REGS_PARM4(ctx))

// --------------------------------------------------------------------

const int key = 0;

// ####################################################################### //
// Structs
// ####################################################################### //

struct go_slice {
    __u64 ptr;
    int len;
    int cap;
};

struct context_id {
  __u32 pid;
  __u64 goid;
};

struct event {
    struct meta_t {
    	__u32 pid;
    	__u64 pos;
    	__u64 buf_size;
    	__u64 msg_size;
    } meta;
	char msg[MAX_PER_CPU_EL_SIZE];
};

// ####################################################################### //
// BPF Maps
// ####################################################################### //

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct context_id);
    __type(value, struct go_slice);
    __uint(max_entries, 1024);
} go_tls_context SEC(".maps");

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

// Get the context ID from the current process and goroutine.
// Currently we are truncating the goid to 32 bits to fit into the context ID that acts as a key in the map.
static __inline struct context_id get_context_id(struct pt_regs* ctx) {
    // Get the current process ID
    __u64 id = bpf_get_current_pid_tgid();
    __u32 tgid = id >> 32;

    // Get the goroutine ID
    __u64 goid;
    __u64 g_ptr = (__u64)GO_G_STRUCT_PTR(ctx);
    bpf_probe_read_user(&goid, sizeof(__u64), (void*)(g_ptr + GOROUTINE_ID_OFFSET));

    // Combine the process ID and the goroutine ID to get a unique context ID.
    struct context_id ctxid = {
        .pid = tgid,
        .goid = goid
    };
    return ctxid;
}

// Read and submit the buffer data - split into chunks if necessary.
static __inline void read_buffer(struct pt_regs *ctx, struct go_slice buf) {
    bpf_printk("reading: %x %d \n", buf.ptr, buf.len);

    // Get the TGID (process ID) for the event
    struct context_id ctx_id = get_context_id(ctx);

    __u64 bytes_sent = 0;
    unsigned int i;

    // Initialize the event from the map.
    struct event *event = bpf_map_lookup_elem(&events_map, &key);
    if (event == NULL) return;

    #pragma unroll
    for (i = 0; i < MAX_CHUNKS; ++i) {
        const __u64 bytes_remaining = buf.len - bytes_sent;

        // Calculate the size to read
        __u64 size_to_read = buf.len;
        if (size_to_read > MAX_PER_CPU_EL_SIZE) size_to_read = MAX_PER_CPU_EL_SIZE;

        event->meta.pid = ctx_id.goid;
        event->meta.pos = bytes_sent;
        event->meta.buf_size = buf.len;
        event->meta.msg_size = size_to_read;

        bpf_probe_read(event->msg, size_to_read, (void *)buf.ptr);

        // Submit the event to the event array
        bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, event, sizeof(*event));

        if (size_to_read == bytes_remaining) break;
        bytes_sent += size_to_read;
    }
}

// Function signature: func (c *Conn) Write(b []byte) (int, error)
// Function symbol:    crypto/tls.(*Conn).Write
SEC("uprobe/go_tls_write_enter")
int go_tls_write_enter(struct pt_regs *ctx) {
    // Get the context ID.
    struct context_id ctx_id = get_context_id(ctx);
    bpf_printk("uprobe invoked on crypto/tls connection write enter with goid: %d", ctx_id.goid);

    // Get the buffer slice as a struct from the registers.
    struct go_slice buf = {
        .ptr = GO_TLS_BUFFER_PTR(ctx),
        .len = GO_TLS_BUFFER_LEN(ctx),
        .cap = GO_TLS_BUFFER_CAP(ctx)
    };

    // Read the buffer data.
    read_buffer(ctx, buf);
    return 0;
}

// Function signature: func (c *Conn) Read(b []byte) (int, error)
// Function symbol:    crypto/tls.(*Conn).Read
SEC("uprobe/go_tls_read_enter")
int gotls_read_enter(struct pt_regs *ctx) {
    // Get the context ID.
    struct context_id ctx_id = get_context_id(ctx);

    bpf_printk("uprobe invoked on crypto/tls connection read enter with goid: %d", ctx_id.goid);

    // Get the buffer slice as a struct from the registers.
    struct go_slice buf = {
        .ptr = GO_TLS_BUFFER_PTR(ctx),
        .len = GO_TLS_BUFFER_LEN(ctx),
        .cap = GO_TLS_BUFFER_CAP(ctx)
    };

    // Save the buffer data (specifically the pointer).
    long err = bpf_map_update_elem(&go_tls_context, &ctx_id, &buf, BPF_ANY);
    if (err != 0) {
        bpf_printk("error saving buffer data: %d\n", err);
    }

    return 0;
}

// Function signature: func (c *Conn) Read(b []byte) (int, error)
// Function symbol:    crypto/tls.(*Conn).Read
SEC("uprobe/go_tls_read_return")
int gotls_read_return(struct pt_regs *ctx) {
    // Get the context ID.
    struct context_id ctx_id = get_context_id(ctx);

    // Get the buffer slice from the map - populated in the enter probe.
    struct go_slice *buf_ptr;
    buf_ptr = bpf_map_lookup_elem(&go_tls_context, &ctx_id);
    if (buf_ptr == NULL) return 0;

    bpf_printk("uprobe invoked on crypto/tls connection read return with goid: %d", ctx_id.goid);

    // Create a new struct from the pointer and the return value.
    struct go_slice buf = {
        .ptr = buf_ptr->ptr,
        .len = (int)PT_REGS_PARM1(ctx),
        .cap = 0
    };

    // Delete the buffer data from the map.
    bpf_map_delete_elem(&go_tls_context, &ctx_id);

    // Read the buffer data.
    read_buffer(ctx, buf);
    return 0;
}

char __license[] SEC("license") = "Dual MIT/GPL";
