/*
 * chan.c — bounded MPMC channel implementation.
 *
 * Structure: circular buffer + one mutex + two condition variables
 * (not-full, not-empty). Every element is copied through memcpy so
 * the caller's value is not aliased; this matches reference §17.2's
 * "send transfers a value" semantics.
 *
 * Closed-channel semantics:
 *   - After close, send returns FUSE_CHAN_CLOSED.
 *   - recv drains buffered values first, then returns
 *     FUSE_CHAN_CLOSED once the buffer is empty.
 *
 * W16 implements capacity=0 as capacity=1 (one-slot buffer) so the
 * rendezvous shape still exercises blocking-on-full and
 * blocking-on-empty. True synchronous rendezvous semantics are
 * W22 concern.
 */

#include "fuse_rt.h"

#include <stdlib.h>
#include <string.h>

#if defined(_WIN32)
#include <windows.h>
#else
#include <pthread.h>
#endif

typedef struct fuse_chan {
    int64_t capacity;
    int64_t elem_bytes;
    uint8_t *buf;       /* contiguous capacity * elem_bytes bytes */
    int64_t head;       /* index of next element to recv */
    int64_t tail;       /* index of next slot to send into */
    int64_t count;      /* number of buffered elements */
    int32_t closed;     /* 0 = open, 1 = closed */

#if defined(_WIN32)
    CRITICAL_SECTION cs;
    CONDITION_VARIABLE not_full;
    CONDITION_VARIABLE not_empty;
#else
    pthread_mutex_t m;
    pthread_cond_t  not_full;
    pthread_cond_t  not_empty;
#endif
} fuse_chan;

/* Thin locking helpers so the same code body handles both
   platforms. */
static void chan_lock(fuse_chan *c) {
#if defined(_WIN32)
    EnterCriticalSection(&c->cs);
#else
    pthread_mutex_lock(&c->m);
#endif
}

static void chan_unlock(fuse_chan *c) {
#if defined(_WIN32)
    LeaveCriticalSection(&c->cs);
#else
    pthread_mutex_unlock(&c->m);
#endif
}

static void chan_wait_not_full(fuse_chan *c) {
#if defined(_WIN32)
    SleepConditionVariableCS(&c->not_full, &c->cs, INFINITE);
#else
    pthread_cond_wait(&c->not_full, &c->m);
#endif
}

static void chan_wait_not_empty(fuse_chan *c) {
#if defined(_WIN32)
    SleepConditionVariableCS(&c->not_empty, &c->cs, INFINITE);
#else
    pthread_cond_wait(&c->not_empty, &c->m);
#endif
}

static void chan_notify_not_full(fuse_chan *c) {
#if defined(_WIN32)
    WakeConditionVariable(&c->not_full);
#else
    pthread_cond_signal(&c->not_full);
#endif
}

static void chan_notify_not_empty(fuse_chan *c) {
#if defined(_WIN32)
    WakeConditionVariable(&c->not_empty);
#else
    pthread_cond_signal(&c->not_empty);
#endif
}

static void chan_notify_all_not_full(fuse_chan *c) {
#if defined(_WIN32)
    WakeAllConditionVariable(&c->not_full);
#else
    pthread_cond_broadcast(&c->not_full);
#endif
}

static void chan_notify_all_not_empty(fuse_chan *c) {
#if defined(_WIN32)
    WakeAllConditionVariable(&c->not_empty);
#else
    pthread_cond_broadcast(&c->not_empty);
#endif
}

void *fuse_rt_chan_new(int64_t capacity, int64_t elem_bytes) {
    if (elem_bytes <= 0) {
        return NULL;
    }
    /* Capacity 0 is the rendezvous shape; we represent it as 1 for W16. */
    int64_t effective = capacity <= 0 ? 1 : capacity;
    fuse_chan *c = (fuse_chan *)calloc(1, sizeof(fuse_chan));
    if (c == NULL) {
        return NULL;
    }
    c->capacity   = effective;
    c->elem_bytes = elem_bytes;
    c->buf        = (uint8_t *)calloc((size_t)effective, (size_t)elem_bytes);
    if (c->buf == NULL) {
        free(c);
        return NULL;
    }
#if defined(_WIN32)
    InitializeCriticalSection(&c->cs);
    InitializeConditionVariable(&c->not_full);
    InitializeConditionVariable(&c->not_empty);
#else
    if (pthread_mutex_init(&c->m, NULL) != 0 ||
        pthread_cond_init(&c->not_full, NULL) != 0 ||
        pthread_cond_init(&c->not_empty, NULL) != 0) {
        free(c->buf);
        free(c);
        return NULL;
    }
#endif
    return c;
}

/* Move one element from `src` into the slot at tail, advancing
   tail. Caller must hold the lock and must have ensured count <
   capacity before calling. */
static void chan_enqueue(fuse_chan *c, const void *src) {
    memcpy(c->buf + (size_t)(c->tail * c->elem_bytes), src, (size_t)c->elem_bytes);
    c->tail = (c->tail + 1) % c->capacity;
    c->count++;
}

/* Move one element from the slot at head into `dst`, advancing
   head. Caller must hold the lock and must have ensured count > 0
   before calling. */
static void chan_dequeue(fuse_chan *c, void *dst) {
    memcpy(dst, c->buf + (size_t)(c->head * c->elem_bytes), (size_t)c->elem_bytes);
    c->head = (c->head + 1) % c->capacity;
    c->count--;
}

int32_t fuse_rt_chan_send(void *chan, const void *value) {
    if (chan == NULL || value == NULL) {
        fuse_rt_abort("fuse_rt_chan_send: null arg");
    }
    fuse_chan *c = (fuse_chan *)chan;
    chan_lock(c);
    while (c->count == c->capacity && !c->closed) {
        chan_wait_not_full(c);
    }
    if (c->closed) {
        chan_unlock(c);
        return FUSE_CHAN_CLOSED;
    }
    chan_enqueue(c, value);
    chan_notify_not_empty(c);
    chan_unlock(c);
    return FUSE_CHAN_OK;
}

int32_t fuse_rt_chan_recv(void *chan, void *out) {
    if (chan == NULL || out == NULL) {
        fuse_rt_abort("fuse_rt_chan_recv: null arg");
    }
    fuse_chan *c = (fuse_chan *)chan;
    chan_lock(c);
    while (c->count == 0 && !c->closed) {
        chan_wait_not_empty(c);
    }
    if (c->count == 0 && c->closed) {
        chan_unlock(c);
        return FUSE_CHAN_CLOSED;
    }
    chan_dequeue(c, out);
    chan_notify_not_full(c);
    chan_unlock(c);
    return FUSE_CHAN_OK;
}

int32_t fuse_rt_chan_try_send(void *chan, const void *value) {
    if (chan == NULL || value == NULL) {
        fuse_rt_abort("fuse_rt_chan_try_send: null arg");
    }
    fuse_chan *c = (fuse_chan *)chan;
    chan_lock(c);
    int32_t rc;
    if (c->closed) {
        rc = FUSE_CHAN_CLOSED;
    } else if (c->count == c->capacity) {
        rc = FUSE_CHAN_WOULD_BLOCK;
    } else {
        chan_enqueue(c, value);
        chan_notify_not_empty(c);
        rc = FUSE_CHAN_OK;
    }
    chan_unlock(c);
    return rc;
}

int32_t fuse_rt_chan_try_recv(void *chan, void *out) {
    if (chan == NULL || out == NULL) {
        fuse_rt_abort("fuse_rt_chan_try_recv: null arg");
    }
    fuse_chan *c = (fuse_chan *)chan;
    chan_lock(c);
    int32_t rc;
    if (c->count > 0) {
        chan_dequeue(c, out);
        chan_notify_not_full(c);
        rc = FUSE_CHAN_OK;
    } else if (c->closed) {
        rc = FUSE_CHAN_CLOSED;
    } else {
        rc = FUSE_CHAN_WOULD_BLOCK;
    }
    chan_unlock(c);
    return rc;
}

void fuse_rt_chan_close(void *chan) {
    if (chan == NULL) {
        return;
    }
    fuse_chan *c = (fuse_chan *)chan;
    chan_lock(c);
    c->closed = 1;
    /* Wake every blocked sender and receiver so they can observe
       the closed state instead of blocking forever. */
    chan_notify_all_not_full(c);
    chan_notify_all_not_empty(c);
    chan_unlock(c);
}

void fuse_rt_chan_free(void *chan) {
    if (chan == NULL) {
        return;
    }
    fuse_chan *c = (fuse_chan *)chan;
#if defined(_WIN32)
    DeleteCriticalSection(&c->cs);
    /* CONDITION_VARIABLE has no Destroy — just free the containing memory. */
#else
    pthread_mutex_destroy(&c->m);
    pthread_cond_destroy(&c->not_full);
    pthread_cond_destroy(&c->not_empty);
#endif
    free(c->buf);
    free(c);
}
