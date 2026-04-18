/*
 * test_sync.c — mutex + condvar + channel smoke tests.
 *
 * Three scenarios:
 *   1. Mutex mutual exclusion: N threads each increment a shared
 *      counter 1000 times under a mutex; final count must equal
 *      N*1000.
 *   2. Condvar wakeup: main waits on a condvar until a worker
 *      sets a flag and signals; the recv side must observe the
 *      set flag on wakeup.
 *   3. Channel round-trip: a worker sends five values; main
 *      receives them in order.
 */

#include "fuse_rt.h"

#include <stdio.h>
#include <stdlib.h>

#define THREADS 4
#define ITERS   1000

static void *shared_mutex;
static int64_t shared_counter;

static int64_t mutex_worker(void *arg) {
    (void)arg;
    for (int i = 0; i < ITERS; i++) {
        fuse_rt_mutex_lock(shared_mutex);
        shared_counter++;
        fuse_rt_mutex_unlock(shared_mutex);
    }
    return 0;
}

static void check(int cond, const char *msg) {
    if (!cond) {
        fprintf(stderr, "test_sync: FAIL %s\n", msg);
        exit(1);
    }
}

static void test_mutex(void) {
    shared_mutex = fuse_rt_mutex_new();
    check(shared_mutex != NULL, "mutex_new returned NULL");
    shared_counter = 0;
    void *handles[THREADS];
    for (int i = 0; i < THREADS; i++) {
        handles[i] = fuse_rt_thread_spawn(mutex_worker, NULL);
        check(handles[i] != NULL, "spawn returned NULL");
    }
    for (int i = 0; i < THREADS; i++) {
        fuse_rt_thread_join(handles[i]);
    }
    check(shared_counter == (int64_t)THREADS * ITERS, "mutex lost an increment");
    fuse_rt_mutex_free(shared_mutex);
}

/* Condvar scenario. */
static void *cv_mutex;
static void *cv_cond;
static int cv_ready;

static int64_t cv_worker(void *arg) {
    (void)arg;
    fuse_rt_sleep_ns(2 * 1000000); /* 2ms so main gets into wait first */
    fuse_rt_mutex_lock(cv_mutex);
    cv_ready = 1;
    fuse_rt_cond_notify_one(cv_cond);
    fuse_rt_mutex_unlock(cv_mutex);
    return 0;
}

static void test_cond(void) {
    cv_mutex = fuse_rt_mutex_new();
    cv_cond  = fuse_rt_cond_new();
    check(cv_mutex != NULL && cv_cond != NULL, "cond new returned NULL");
    cv_ready = 0;
    void *h = fuse_rt_thread_spawn(cv_worker, NULL);
    check(h != NULL, "spawn returned NULL");
    fuse_rt_mutex_lock(cv_mutex);
    while (!cv_ready) {
        fuse_rt_cond_wait(cv_cond, cv_mutex);
    }
    fuse_rt_mutex_unlock(cv_mutex);
    fuse_rt_thread_join(h);
    check(cv_ready == 1, "cond wait returned without ready flag set");
    fuse_rt_cond_free(cv_cond);
    fuse_rt_mutex_free(cv_mutex);
}

/* Channel scenario. */
static void *chan_handle;

static int64_t chan_worker(void *arg) {
    (void)arg;
    for (int64_t i = 1; i <= 5; i++) {
        int64_t value = i * 10;
        fuse_rt_chan_send(chan_handle, &value);
    }
    fuse_rt_chan_close(chan_handle);
    return 0;
}

static void test_chan(void) {
    chan_handle = fuse_rt_chan_new(2, (int64_t)sizeof(int64_t));
    check(chan_handle != NULL, "chan_new returned NULL");
    void *h = fuse_rt_thread_spawn(chan_worker, NULL);
    check(h != NULL, "spawn returned NULL");
    for (int64_t i = 1; i <= 5; i++) {
        int64_t got = 0;
        int32_t rc = fuse_rt_chan_recv(chan_handle, &got);
        check(rc == FUSE_CHAN_OK, "recv before close should succeed");
        check(got == i * 10, "received wrong value");
    }
    /* Draining a closed channel returns FUSE_CHAN_CLOSED. */
    int64_t sink = 0;
    int32_t rc = fuse_rt_chan_recv(chan_handle, &sink);
    check(rc == FUSE_CHAN_CLOSED, "recv on drained+closed should return CLOSED");

    /* Send on closed chan returns CLOSED. */
    int64_t x = 99;
    rc = fuse_rt_chan_send(chan_handle, &x);
    check(rc == FUSE_CHAN_CLOSED, "send on closed should return CLOSED");

    /* try_send / try_recv on closed. */
    rc = fuse_rt_chan_try_send(chan_handle, &x);
    check(rc == FUSE_CHAN_CLOSED, "try_send on closed should return CLOSED");
    rc = fuse_rt_chan_try_recv(chan_handle, &sink);
    check(rc == FUSE_CHAN_CLOSED, "try_recv on drained+closed should return CLOSED");

    fuse_rt_thread_join(h);
    fuse_rt_chan_free(chan_handle);
}

int main(void) {
    test_mutex();
    test_cond();
    test_chan();
    printf("test_sync: ok\n");
    return 0;
}
