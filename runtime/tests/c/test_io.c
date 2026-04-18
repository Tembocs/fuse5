/*
 * test_io.c — fuse_rt_write_stdout / fuse_rt_write_stderr.
 *
 * Writes a known byte sequence to each stream and checks the
 * return value. Does not attempt to capture the streams; the real
 * proof is the byte count matching the expected length.
 */

#include "fuse_rt.h"

#include <stdio.h>
#include <string.h>
#include <stdlib.h>

static void check(int cond, const char *msg) {
    if (!cond) {
        fprintf(stderr, "test_io: FAIL %s\n", msg);
        exit(1);
    }
}

int main(void) {
    const char *hello = "test_io: hello\n";
    int64_t len = (int64_t)strlen(hello);

    int64_t wrote = fuse_rt_write_stdout((const uint8_t *)hello, len);
    check(wrote == len, "stdout wrote != len");

    wrote = fuse_rt_write_stderr((const uint8_t *)hello, len);
    check(wrote == len, "stderr wrote != len");

    /* Zero-length write is a no-op and returns 0. */
    wrote = fuse_rt_write_stdout((const uint8_t *)hello, 0);
    check(wrote == 0, "zero-len stdout should return 0");

    printf("test_io: ok\n");
    return 0;
}
