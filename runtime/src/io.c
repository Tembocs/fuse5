/*
 * io.c — standard-stream byte writes.
 *
 * Best-effort: short writes loop until the full buffer is emitted
 * or the host returns -1. Returns the number of bytes successfully
 * written; -1 on error.
 */

#include "fuse_rt.h"

#include <stdio.h>

static int64_t write_stream(FILE *stream, const uint8_t *bytes, int64_t len) {
    if (bytes == NULL || len <= 0) {
        return 0;
    }
    int64_t written = 0;
    while (written < len) {
        size_t remaining = (size_t)(len - written);
        size_t n = fwrite(bytes + written, 1, remaining, stream);
        if (n == 0) {
            if (ferror(stream)) {
                return -1;
            }
            break;
        }
        written += (int64_t)n;
    }
    fflush(stream);
    return written;
}

int64_t fuse_rt_write_stdout(const uint8_t *bytes, int64_t len) {
    return write_stream(stdout, bytes, len);
}

int64_t fuse_rt_write_stderr(const uint8_t *bytes, int64_t len) {
    return write_stream(stderr, bytes, len);
}
