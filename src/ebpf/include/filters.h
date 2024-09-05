#pragma once

#define HOST_HEADER "Host: "
#define AUTH_HEADER "Authorization: "
#define HOST_HEADER_LEN 6
#define AUTH_HEADER_LEN 14
#define MAX_HEADER_LENGTH 63

struct http_request_t {
    // Request headers
    char host[MAX_HEADER_LENGTH];
    char auth[MAX_HEADER_LENGTH];

    // Request data
    __u8 *data;
    int data_len;

    // Internal state
    __u32 line_start;
    char cur_line[MAX_HEADER_LENGTH];
};

//struct {
//    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
//    __type(key, __u32);
//    __type(value, struct http_request_t);
//    __uint(max_entries, 1);
//} http_request_map SEC(".maps");

static int parse_headers(u32 index, struct http_request_t *req) {
    // Stop parsing if we've read enough characters
    if (index > MAX_CHUNK_SIZE || index >= req->data_len) {
        return 1;
    }

    // Safe read the current byte
    __u8 byte = 0;
    if (index >= 0 && index < req->data_len) {
        bpf_probe_read(&byte, sizeof(byte), req->data + index);
    }

    // Check if we've reached the end of a line or max header length
    if (byte == '\n') {
        __u32 line_len = (index - req->line_start) & MAX_HEADER_LENGTH;

        // Safe read the current line we found and save it in the request context
        bpf_probe_read(&req->cur_line, line_len, req->data + req->line_start);

        if (line_len >= HOST_HEADER_LEN && !__builtin_memcmp(req->cur_line, HOST_HEADER, HOST_HEADER_LEN)) {
            // Copy Host header
            bpf_probe_read(&req->host, line_len, &req->cur_line);
        } else if (line_len >= AUTH_HEADER_LEN && !__builtin_memcmp(req->cur_line, AUTH_HEADER, AUTH_HEADER_LEN)) {
            // Copy Authorization header
            bpf_probe_read(req->auth, line_len, req->cur_line);
        }

        // Move to the next line
        if (index + 1 < req->data_len) {
            req->line_start = index + 1;
        }
    }

    return 0;
}


static __inline void parse_request(struct ssl_event_t *event) {
    // Initialize the event from the map.
//    struct http_request_t *request = bpf_map_lookup_elem(&http_request_map, &ZERO);
//    if (request == NULL){
//        bpf_printk("error creating http_request_t");
//        return;
//    }

    struct http_request_t request = {
        .data = event->data,
        .data_len = event->meta.data_size,
        .line_start = 0
    };

    // Parse the headers
    bpf_loop(MAX_CHUNK_SIZE, parse_headers, &request, 0);

    // Print the host using bpf_printk
    bpf_printk("Host: %c%c%c%c%c%c", request.host[0], request.host[1], request.host[2], request.host[3], request.host[4], request.host[5]);

//    bpf_printk("Host: %s", request.host);
//    bpf_printk("Authorization: %s", request.auth);
}

static __inline bool startsWith(__u8 *line, int line_len, __u8 *target, int target_len) {
    // If the line is shorter than the target, it cannot start with the target
    if (line_len < target_len) return false;

    // Compare each byte of the target with the beginning of the line
    for (int k = 0; k < target_len; k++) {
        if (line[k] != target[k]) return false;
    }

    return true;
}

static __inline bool containsString(__u8 *line, int line_len, __u8 *target, int target_len) {
    // If the line is shorter than the target, it cannot contain the target
    if (line_len < target_len) return false;

    // Compare each substring of the line with the target
    // Cannot return from the for loop - it causes ebpf verifier error
    bool match;
    for (int i = 0; i <= line_len - target_len; i++) {
        match = true;
        for (int j = 0; j < target_len; j++) {
            if (line[i + j] != target[j]) {
                match = false;
                break;
            }
        }
        if (match) break;
    }

    return match;
}

static __inline bool isHostHeader(__u8 *line, int line_len) {
    __u8 target[] = {'H', 'o', 's', 't', ':'};
    return startsWith(line, line_len, target, 5);
}

static __inline bool isAuthHeader(__u8 *line, int line_len) {
    __u8 target[] = {'A', 'u', 't', 'h', 'o', 'r', 'i', 'z', 'a', 't', 'i', 'o', 'n', ':'};
    return startsWith(line, line_len, target, 14);
}

static __inline bool isAwsApiCall(__u8 *line, int line_len) {
    // AWS requests are authed using AWS4-HMAC-SHA256
    __u8 target[] = {'A', 'W', 'S', '4', '-', 'H', 'M', 'A', 'a'};
    return startsWith(line + 15, line_len - 15, target, sizeof(target));
}

static __inline bool shouldSendEvent(struct ssl_event_t *event) {
//    for (int i = 0; i < event->meta.data_size && i < MAX_CHARS_TO_READ; i++) {
//        if (event->data[i] == '\n' || i - start >= MAX_LINE_LENGTH) {
//            line_len = i - start;
//            start = i + 1;  // Move to the next line
//
//            if(isAwsApiCall(event->data + start, line_len)) return true;
//        }
//    }

    parse_request(event);

    return false;
}