#pragma once

#define HOST_HEADER_LEN 6
#define AUTH_HEADER_LEN 14
#define MAX_HEADER_LENGTH 255

#define HOST_HEADER "Host: "
#define AUTH_HEADER "Authorization: "

#define HOST_AWS "amazonaws.com"
#define HOST_AWS_LEN 13

struct http_request_t {
    // Request headers
    __u32 host_len;
    char host[MAX_HEADER_LENGTH];

    __u32 auth_len;
    char auth[MAX_HEADER_LENGTH];

    // Internal state
    char cur_line[MAX_HEADER_LENGTH];
};

struct http_request_ctx_t {
    __u8 *data;
    int data_len;
    __u32 line_start;
};

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __type(key, __u32);
    __type(value, struct http_request_t);
    __uint(max_entries, 1);
} http_request_map SEC(".maps");


// ####################################################################### //
// Function declarations
// ####################################################################### //

static __inline bool should_send_event(struct ssl_event_t *event);
