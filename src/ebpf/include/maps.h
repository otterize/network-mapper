const __u32 MAX_SIZE = 4096;
const __u32 MAX_ENTRIES_HASH = 4096;

struct ssl_event_meta_t {
    __u32 pid;
    __u64 timestamp;
    __u32 dataSize;
};

struct ssl_event_t {
    struct ssl_event_meta_t meta;
    __u8 data[MAX_SIZE];
};

struct ssl_context_t {
    __u64 size;
    __u64 buffer;
};

struct target_t {
    _Bool enabled;
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
    __type(value, struct target_t);
} targets SEC(".maps");
