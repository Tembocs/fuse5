/*
 * time.c — monotonic + wall clocks and sleep.
 */

#include "fuse_rt.h"

#if defined(_WIN32)
#include <windows.h>
#include <profileapi.h>
#else
#include <time.h>
#endif

int64_t fuse_rt_monotonic_ns(void) {
#if defined(_WIN32)
    LARGE_INTEGER freq, now;
    QueryPerformanceFrequency(&freq);
    QueryPerformanceCounter(&now);
    /* Avoid 64-bit overflow: split multiply. */
    int64_t secs = now.QuadPart / freq.QuadPart;
    int64_t rem  = now.QuadPart % freq.QuadPart;
    return secs * 1000000000 + (rem * 1000000000) / freq.QuadPart;
#else
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (int64_t)ts.tv_sec * 1000000000 + (int64_t)ts.tv_nsec;
#endif
}

int64_t fuse_rt_wall_ns(void) {
#if defined(_WIN32)
    FILETIME ft;
    GetSystemTimeAsFileTime(&ft);
    /* FILETIME is 100-nanosecond intervals since 1601-01-01.
       Convert to nanoseconds since 1970-01-01. */
    ULARGE_INTEGER u;
    u.LowPart  = ft.dwLowDateTime;
    u.HighPart = ft.dwHighDateTime;
    const int64_t WIN_EPOCH_OFFSET_100NS = 116444736000000000LL;
    int64_t hundreds = (int64_t)u.QuadPart - WIN_EPOCH_OFFSET_100NS;
    return hundreds * 100;
#else
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return (int64_t)ts.tv_sec * 1000000000 + (int64_t)ts.tv_nsec;
#endif
}

void fuse_rt_sleep_ns(int64_t nanos) {
    if (nanos <= 0) {
        return;
    }
#if defined(_WIN32)
    /* Sleep takes milliseconds; round up so we never sleep less. */
    DWORD ms = (DWORD)((nanos + 999999) / 1000000);
    Sleep(ms);
#else
    struct timespec ts;
    ts.tv_sec  = (time_t)(nanos / 1000000000);
    ts.tv_nsec = (long)(nanos % 1000000000);
    nanosleep(&ts, NULL);
#endif
}
