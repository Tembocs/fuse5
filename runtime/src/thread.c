/*
 * thread.c — cross-platform OS thread surface.
 *
 * Win32 on Windows (CreateThread / WaitForSingleObject /
 * CloseHandle); pthreads on POSIX. The handle we return to the
 * compiler is an opaque pointer to a fuse_thread struct that owns
 * the platform-specific handle plus a result slot the entry
 * function writes before exit.
 */

#include "fuse_rt.h"

#include <stdlib.h>
#include <string.h>

#if defined(_WIN32)
#include <windows.h>
#include <process.h>
#else
#include <pthread.h>
#include <sched.h>
#include <unistd.h>
#endif

typedef struct fuse_thread {
    int64_t (*entry)(void *);
    void *arg;
    int64_t result;
#if defined(_WIN32)
    HANDLE handle;
    DWORD  tid;
#else
    pthread_t handle;
#endif
} fuse_thread;

#if defined(_WIN32)
static unsigned __stdcall thread_trampoline(void *raw) {
    fuse_thread *t = (fuse_thread *)raw;
    t->result = t->entry(t->arg);
    return 0;
}
#else
static void *thread_trampoline(void *raw) {
    fuse_thread *t = (fuse_thread *)raw;
    t->result = t->entry(t->arg);
    return NULL;
}
#endif

void *fuse_rt_thread_spawn(int64_t (*entry)(void *), void *arg) {
    if (entry == NULL) {
        return NULL;
    }
    fuse_thread *t = (fuse_thread *)calloc(1, sizeof(fuse_thread));
    if (t == NULL) {
        return NULL;
    }
    t->entry = entry;
    t->arg   = arg;
#if defined(_WIN32)
    /* _beginthreadex is preferred over CreateThread because it
       initialises the CRT's per-thread state (errno, etc). */
    uintptr_t raw = _beginthreadex(NULL, 0, thread_trampoline, t, 0, (unsigned *)&t->tid);
    if (raw == 0) {
        free(t);
        return NULL;
    }
    t->handle = (HANDLE)raw;
#else
    if (pthread_create(&t->handle, NULL, thread_trampoline, t) != 0) {
        free(t);
        return NULL;
    }
#endif
    return t;
}

int64_t fuse_rt_thread_join(void *handle) {
    if (handle == NULL) {
        fuse_rt_abort("fuse_rt_thread_join: null handle");
    }
    fuse_thread *t = (fuse_thread *)handle;
#if defined(_WIN32)
    WaitForSingleObject(t->handle, INFINITE);
    CloseHandle(t->handle);
#else
    pthread_join(t->handle, NULL);
#endif
    int64_t result = t->result;
    free(t);
    return result;
}

int64_t fuse_rt_thread_id(void) {
#if defined(_WIN32)
    return (int64_t)GetCurrentThreadId();
#else
    /* pthread_self returns an opaque type; we cast via uintptr_t so
       the id is a stable number within a process even if the
       underlying pthread_t is not an integer. */
    return (int64_t)(uintptr_t)pthread_self();
#endif
}

void fuse_rt_thread_yield(void) {
#if defined(_WIN32)
    SwitchToThread();
#else
    sched_yield();
#endif
}
