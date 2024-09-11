#include "filters.h"

static __inline bool should_send_event(struct ssl_event_t *event) {
    // for now, we do not do any filtering in eBPF, since
    // it caused us issues, and we can do it in userspace.
    // This function is left for future use. Please
    // call it before sending events.
    return true;
}