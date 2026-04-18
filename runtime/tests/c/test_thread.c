/*
 * test_thread.c — fuse_rt_thread_spawn / join / id.
 *
 * Spawns a worker that squares its input, joins, and confirms the
 * returned value matches. Also confirms the worker's thread id
 * differs from the main thread's.
 */

#include "fuse_rt.h"

#include <stdio.h>
#include <stdlib.h>

static int64_t worker_tid;

static int64_t worker_entry(void *arg) {
    int64_t n = (int64_t)(intptr_t)arg;
    worker_tid = fuse_rt_thread_id();
    return n * n;
}

static void check(int cond, const char *msg) {
    if (!cond) {
        fprintf(stderr, "test_thread: FAIL %s\n", msg);
        exit(1);
    }
}

int main(void) {
    int64_t main_tid = fuse_rt_thread_id();
    check(main_tid > 0, "main thread id should be non-zero");

    void *h = fuse_rt_thread_spawn(worker_entry, (void *)(intptr_t)7);
    check(h != NULL, "spawn returned NULL");
    int64_t result = fuse_rt_thread_join(h);
    check(result == 49, "worker did not return 7*7");
    check(worker_tid != main_tid, "worker and main share a thread id");

    /* Yield does not crash. */
    fuse_rt_thread_yield();

    printf("test_thread: ok (main=%lld worker=%lld)\n",
           (long long)main_tid, (long long)worker_tid);
    return 0;
}
