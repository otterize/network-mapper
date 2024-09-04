//go:build ignore

#include "headers.h"
#include "maps.h"
#include "common.h"

// ####################################################################### //
// Definitions
// ####################################################################### //

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

// ####################################################################### //
// Functions
// ####################################################################### //

// Get the context ID from the current process and goroutine.
// Currently we are truncating the goid to 32 bits to fit into the context ID that acts as a key in the map.
static __inline struct go_context_id_t get_context_id(struct pt_regs* ctx) {
    // Get the current process ID
    __u64 id = bpf_get_current_pid_tgid();
    __u64 tgid = id >> 32;

    // Get the goroutine ID
    __u64 goid;
    __u64 g_ptr = (__u64)GO_G_STRUCT_PTR(ctx);
    bpf_probe_read_user(&goid, sizeof(__u64), (void*)(g_ptr + GOROUTINE_ID_OFFSET));

    // Combine the process ID and the goroutine ID to get a unique context ID.
    struct go_context_id_t ctxid = {
        .pid = tgid,
        .goid = goid
    };
    return ctxid;
}

// Read and submit the buffer data - split into chunks if necessary.
static __inline void read_buffer(struct pt_regs *ctx, struct go_slice_t *buf, enum direction_t direction) {
    bpf_printk("reading: %x %d", buf->ptr, buf->len);

    __u64 bytes_sent = 0;
    unsigned int i;

    #pragma unroll
    for (i = 0; i < MAX_CHUNKS; ++i) {
        // Calculate the size to read
        __u64 size_to_read = buf->len;
        if (size_to_read > MAX_DATA_SIZE) size_to_read = MAX_DATA_SIZE;

        send_event(ctx, buf->ptr, size_to_read, buf->len, direction);

        const __u64 bytes_remaining = buf->len - bytes_sent;
        if (size_to_read >= bytes_remaining) break;
        bytes_sent += size_to_read;
    }
}

// Function signature: func (c *Conn) Write(b []byte) (int, error)
// Function symbol:    crypto/tls.(*Conn).Write
SEC("uprobe/go_tls_write_enter")
int go_tls_write_enter(struct pt_regs *ctx) {
//    if (!shouldTrace()) return 0;

    // Get the context ID.
    struct go_context_id_t ctx_id = get_context_id(ctx);
    bpf_printk("uprobe invoked on crypto/tls connection write enter with id: %d-%d", ctx_id.pid, ctx_id.goid);

    // Get the buffer slice as a struct from the registers.
    struct go_slice_t buf = {
        .ptr = GO_TLS_BUFFER_PTR(ctx),
        .len = GO_TLS_BUFFER_LEN(ctx),
        .cap = GO_TLS_BUFFER_CAP(ctx)
    };

    // Read the buffer data.
    read_buffer(ctx, &buf, EGRESS);
    return 0;
}

// Function signature: func (c *Conn) Read(b []byte) (int, error)
// Function symbol:    crypto/tls.(*Conn).Read
SEC("uprobe/go_tls_read_enter")
int gotls_read_enter(struct pt_regs *ctx) {
    if (!shouldTrace()) return 0;

    // Get the context ID.
    struct go_context_id_t ctx_id = get_context_id(ctx);

    bpf_printk("uprobe invoked on crypto/tls connection read enter with id: %d-%d", ctx_id.pid, ctx_id.goid);

    // Get the buffer slice as a struct from the registers.
    struct go_slice_t buf = {
        .ptr = GO_TLS_BUFFER_PTR(ctx),
        .len = GO_TLS_BUFFER_LEN(ctx),
        .cap = GO_TLS_BUFFER_CAP(ctx)
    };

    // Save the buffer data (specifically the pointer).
    long err = bpf_map_update_elem(&go_tls_context, &ctx_id, &buf, BPF_ANY);
    if (err) {
        bpf_printk("error saving buffer data: %d with id: %d-%d", err, ctx_id.pid, ctx_id.goid);
    }

    return 0;
}

// Function signature: func (c *Conn) Read(b []byte) (int, error)
// Function symbol:    crypto/tls.(*Conn).Read
SEC("uprobe/go_tls_read_return")
int gotls_read_return(struct pt_regs *ctx) {
    // Get the context ID.
    struct go_context_id_t ctx_id = get_context_id(ctx);

    // Get the buffer slice from the map - populated in the enter probe.
    struct go_slice_t *buf_ptr;
    buf_ptr = bpf_map_lookup_elem(&go_tls_context, &ctx_id);
    if (buf_ptr == NULL){
        bpf_printk("uprobe miss on crypto/tls connection read return with id: %d-%d", ctx_id.pid, ctx_id.goid);
        return 0;
    }

    bpf_printk("uprobe invoked on crypto/tls connection read return with id: %d-%d", ctx_id.pid, ctx_id.goid);

    // Create a new struct from the pointer and the return value.
    struct go_slice_t buf = {
        .ptr = buf_ptr->ptr,
        .len = (int)PT_REGS_PARM1(ctx),
        .cap = 0
    };

    // Delete the buffer data from the map.
    bpf_map_delete_elem(&go_tls_context, &ctx_id);

    // Read the buffer data.
    read_buffer(ctx, &buf, INGRESS);
    return 0;
}
