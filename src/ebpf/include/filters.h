#pragma once

// TODO: reconsider consts
const __u32 MAX_LINE_LENGTH = 100;
const __u32 MAX_CHARS_TO_READ = 1024;

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
    int start = 0;
    int line_len = 0;

    for (int i = 0; i < event->meta.data_size && i < MAX_CHARS_TO_READ; i++) {
        if (event->data[i] == '\n' || i - start >= MAX_LINE_LENGTH) {
            line_len = i - start;
            start = i + 1;  // Move to the next line

            if(isAwsApiCall(event->data + start, line_len)) return true;
        }
    }

    return false;
}