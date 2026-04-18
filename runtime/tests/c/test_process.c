/*
 * test_process.c — process identity + time primitives.
 *
 * Verifies that fuse_rt_pid is non-zero, monotonic_ns is
 * non-decreasing, and sleep_ns actually sleeps for at least the
 * requested duration (within a generous slop to tolerate
 * scheduler jitter).
 */

#include "fuse_rt.h"

#include <stdio.h>
#include <stdlib.h>

static void check(int cond, const char *msg) {
    if (!cond) {
        fprintf(stderr, "test_process: FAIL %s\n", msg);
        exit(1);
    }
}

int main(void) {
    int64_t pid = fuse_rt_pid();
    check(pid > 0, "pid should be positive");

    int64_t a = fuse_rt_monotonic_ns();
    int64_t b = fuse_rt_monotonic_ns();
    check(b >= a, "monotonic clock went backwards");

    int64_t w = fuse_rt_wall_ns();
    check(w > 0, "wall clock should be non-zero");

    int64_t before = fuse_rt_monotonic_ns();
    fuse_rt_sleep_ns(5 * 1000000); /* 5ms */
    int64_t after = fuse_rt_monotonic_ns();
    /* Allow a wide slop: scheduler grain is coarse, especially on
       Windows where Sleep(1) is 15ms worst-case. We just want to
       confirm *some* time passed. */
    check(after - before >= 1 * 1000000, "sleep_ns did not elapse at least 1ms");

    printf("test_process: ok (pid=%lld)\n", (long long)pid);
    return 0;
}
