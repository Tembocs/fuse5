/*
 * process.c — process-level identity and environment queries.
 *
 * W16 scope: pid only. Args and env are W18 concern (CLI surface)
 * and are deferred here so the header stays stable without a
 * half-baked argv plumbing.
 */

#include "fuse_rt.h"

#if defined(_WIN32)
#include <windows.h>
#else
#include <unistd.h>
#endif

int64_t fuse_rt_pid(void) {
#if defined(_WIN32)
    return (int64_t)GetCurrentProcessId();
#else
    return (int64_t)getpid();
#endif
}
