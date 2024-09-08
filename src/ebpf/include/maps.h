#pragma once

// The maximum percpu map element size is 32KB - we set it at 30KB.
// Source: https://github.com/iovisor/bcc/issues/2519
const __u32 MAX_DATA_SIZE = 30720; // 30KB max buffer size

const __u32 MAX_CHUNKS = 4;
const __u32 MAX_CHUNK_SIZE = 4096;
const __u32 MAX_ENTRIES_HASH = 4096;

const int ZERO = 0;

// ####################################################################### //
// Structs
// ####################################################################### //

enum direction_t {
  EGRESS,
  INGRESS,
};

struct ssl_event_meta_t {
    __u32 pid;
    __u64 timestamp;
    __u32 data_size;
    __u32 total_size;
    enum direction_t direction;
};

struct ssl_event_t {
    struct ssl_event_meta_t meta;
    __u8 data[MAX_DATA_SIZE];
};

struct target_t {
    _Bool enabled;
};

struct ssl_context_t {
    __u64 size;
    __u64 buffer;
};

struct go_slice_t {
    __u64 ptr;
    int len;
    int cap;
};

struct go_context_id_t {
  __u64 pid;
  __u64 goid;
};

// ####################################################################### //
// BPF Maps
// ####################################################################### //

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

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, __u64);
    __type(value, struct ssl_context_t);
} ssl_contexts SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct go_context_id_t);
    __type(value, struct go_slice_t);
    __uint(max_entries, 1024);
} go_tls_context SEC(".maps");
