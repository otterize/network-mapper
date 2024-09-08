
#include "filters.h"

static __inline int helper_memcmp(const char *s1, const char *s2, int len) {
    #pragma unroll
    for (int i = 0; i < len; i++) {
        if (s1[i] != s2[i]) {
            return -1;  // Strings do not match
        }
    }
    return 0;  // Strings match
}

static __inline bool is_http_request(__u8 *data, int data_len) {
    int method_lens[] = {3, 4, 3, 6, 4, 7, 6};
    const char *methods[] = {"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"};

    // total size in bytes / size of a single element (char *)
    int num_methods = sizeof(methods) / sizeof(methods[0]);

    // Minimum length to fit a method and space "GET "
    if (data_len < 4) return false;

    // Check for any HTTP method at the start of the data
    #pragma unroll
    for (int i = 0; i < num_methods; i++) {
        int method_len = method_lens[i];
        if (data_len >= method_len + 1 && helper_memcmp((char *)data, methods[i], method_len) == 0 && data[method_len] == ' ') {
            return true;
        }
    }

    return false;
}

static int parse_headers(u32 index, struct http_request_ctx_t *ctx) {
    // Grab the request object from the map.
    struct http_request_t *req = bpf_map_lookup_elem(&http_request_map, &ZERO);
    if (req == NULL){
        bpf_printk("error creating http_request_t");
        return 1;
    }

    // Stop parsing if we've read enough characters
    if (index > MAX_CHUNK_SIZE || index >= ctx->data_len) {
        return 1;
    }

    // Safe read the current byte
    __u8 byte = 0;
    if (index >= 0 && index < ctx->data_len) {
        bpf_probe_read(&byte, sizeof(byte), ctx->data + index);
    }

    // Check if we've reached the end of a line or max header length
    if (byte == '\n') {
        __u32 line_len = (index - ctx->line_start) & MAX_HEADER_LENGTH;

        // Safe read the current line we found and save it in the request object
        bpf_probe_read(&req->cur_line, line_len, ctx->data + ctx->line_start);

        // Check if the line is a Host or Authorization header and save it in the request object
        if (line_len >= HOST_HEADER_LEN && !helper_memcmp(req->cur_line, HOST_HEADER, HOST_HEADER_LEN)) {
            // Copy Host header
            bpf_probe_read(&req->host, line_len, &req->cur_line);
            req->host_len = line_len;
        } else if (line_len >= AUTH_HEADER_LEN && !helper_memcmp(req->cur_line, AUTH_HEADER, AUTH_HEADER_LEN)) {
            // Copy Authorization header
            bpf_probe_read(req->auth, line_len, req->cur_line);
            req->auth_len = line_len;
        }

        // Move to the next line
        if (index + 1 < ctx->data_len) {
            ctx->line_start = index + 1;
        }
    }

    return 0;
}

static __inline void parse_request(struct ssl_event_t *event) {
    struct http_request_ctx_t ctx = {
        .data = event->data,
        .data_len = event->meta.data_size,
        .line_start = 0
    };

    // Parse the headers
    bpf_loop(MAX_CHUNK_SIZE, parse_headers, &ctx, 0);
}

static __inline bool is_aws_api_call(struct http_request_t *req) {
    // Iterate over the host header to look for the substring
    #pragma unroll
    for (int i = 0; i < MAX_HEADER_LENGTH - HOST_AWS_LEN; i++) {
        // Breaking early if we reach the end of the header - putting this here helps the compiler optimize the loop
        if(i >= req->host_len) break;

        // Compare substring
        if (helper_memcmp(&req->host[i], HOST_AWS, HOST_AWS_LEN) == 0) {
            return true;  // domain found
        }
    }

    return false;  // Substring not found
}

static __inline bool should_send_event(struct ssl_event_t *event) {
    bool should_send = false;

    // Handle event which is an HTTP request
    if(is_http_request(event->data, event->meta.data_size)) {
        // Populates the request data in the map
        parse_request(event);

        // Grab the request object from the map.
        struct http_request_t *request = bpf_map_lookup_elem(&http_request_map, &ZERO);
        if (request == NULL){
            bpf_printk("error creating http_request_t");
            return false;
        }

        if(is_aws_api_call(request)) should_send = true;

        bpf_map_delete_elem(&http_request_map, &ZERO);
    }


    return should_send;
}