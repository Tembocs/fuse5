/*
 * sync.c — mutex and condition-variable primitives.
 *
 * Each handle the runtime exposes to the compiler is a heap-
 * allocated wrapper that owns a platform primitive. Allocating
 * on the heap keeps the opaque-handle contract honest: the
 * compiler does not know the size of CRITICAL_SECTION /
 * pthread_mutex_t.
 */

#include "fuse_rt.h"

#include <stdlib.h>

#if defined(_WIN32)
#include <windows.h>
#else
#include <pthread.h>
#endif

typedef struct fuse_mutex {
#if defined(_WIN32)
    CRITICAL_SECTION cs;
#else
    pthread_mutex_t m;
#endif
} fuse_mutex;

typedef struct fuse_cond {
#if defined(_WIN32)
    CONDITION_VARIABLE cv;
#else
    pthread_cond_t c;
#endif
} fuse_cond;

void *fuse_rt_mutex_new(void) {
    fuse_mutex *m = (fuse_mutex *)calloc(1, sizeof(fuse_mutex));
    if (m == NULL) {
        return NULL;
    }
#if defined(_WIN32)
    InitializeCriticalSection(&m->cs);
#else
    if (pthread_mutex_init(&m->m, NULL) != 0) {
        free(m);
        return NULL;
    }
#endif
    return m;
}

void fuse_rt_mutex_lock(void *mutex) {
    if (mutex == NULL) {
        fuse_rt_abort("fuse_rt_mutex_lock: null mutex");
    }
    fuse_mutex *m = (fuse_mutex *)mutex;
#if defined(_WIN32)
    EnterCriticalSection(&m->cs);
#else
    pthread_mutex_lock(&m->m);
#endif
}

void fuse_rt_mutex_unlock(void *mutex) {
    if (mutex == NULL) {
        fuse_rt_abort("fuse_rt_mutex_unlock: null mutex");
    }
    fuse_mutex *m = (fuse_mutex *)mutex;
#if defined(_WIN32)
    LeaveCriticalSection(&m->cs);
#else
    pthread_mutex_unlock(&m->m);
#endif
}

void fuse_rt_mutex_free(void *mutex) {
    if (mutex == NULL) {
        return;
    }
    fuse_mutex *m = (fuse_mutex *)mutex;
#if defined(_WIN32)
    DeleteCriticalSection(&m->cs);
#else
    pthread_mutex_destroy(&m->m);
#endif
    free(m);
}

void *fuse_rt_cond_new(void) {
    fuse_cond *c = (fuse_cond *)calloc(1, sizeof(fuse_cond));
    if (c == NULL) {
        return NULL;
    }
#if defined(_WIN32)
    InitializeConditionVariable(&c->cv);
#else
    if (pthread_cond_init(&c->c, NULL) != 0) {
        free(c);
        return NULL;
    }
#endif
    return c;
}

void fuse_rt_cond_wait(void *cond, void *mutex) {
    if (cond == NULL || mutex == NULL) {
        fuse_rt_abort("fuse_rt_cond_wait: null cond or mutex");
    }
    fuse_cond  *c = (fuse_cond *)cond;
    fuse_mutex *m = (fuse_mutex *)mutex;
#if defined(_WIN32)
    SleepConditionVariableCS(&c->cv, &m->cs, INFINITE);
#else
    pthread_cond_wait(&c->c, &m->m);
#endif
}

void fuse_rt_cond_notify_one(void *cond) {
    if (cond == NULL) {
        fuse_rt_abort("fuse_rt_cond_notify_one: null cond");
    }
    fuse_cond *c = (fuse_cond *)cond;
#if defined(_WIN32)
    WakeConditionVariable(&c->cv);
#else
    pthread_cond_signal(&c->c);
#endif
}

void fuse_rt_cond_notify_all(void *cond) {
    if (cond == NULL) {
        fuse_rt_abort("fuse_rt_cond_notify_all: null cond");
    }
    fuse_cond *c = (fuse_cond *)cond;
#if defined(_WIN32)
    WakeAllConditionVariable(&c->cv);
#else
    pthread_cond_broadcast(&c->c);
#endif
}

void fuse_rt_cond_free(void *cond) {
    if (cond == NULL) {
        return;
    }
    fuse_cond *c = (fuse_cond *)cond;
#if defined(_WIN32)
    /* CONDITION_VARIABLE has no Destroy — the struct is just freed. */
#else
    pthread_cond_destroy(&c->c);
#endif
    free(c);
}
