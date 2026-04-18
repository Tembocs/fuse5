/*
 * abort.c — process-control primitives.
 *
 * fuse_rt_abort / fuse_rt_panic / fuse_rt_exit are the only functions
 * in the runtime that may never return. Every other surface is free
 * to fail gracefully; these three must terminate the process cleanly
 * with a diagnostic.
 */

#include "fuse_rt.h"

#include <stdio.h>
#include <stdlib.h>

_Noreturn void fuse_rt_abort(const char *message) {
    if (message != NULL) {
        fputs("fuse_rt_abort: ", stderr);
        fputs(message, stderr);
        fputc('\n', stderr);
    } else {
        fputs("fuse_rt_abort: (no message)\n", stderr);
    }
    fflush(stderr);
    abort();
}

_Noreturn void fuse_rt_panic(const char *message) {
    fputs("fuse panic: ", stderr);
    if (message != NULL) {
        fputs(message, stderr);
    } else {
        fputs("(no message)", stderr);
    }
    fputc('\n', stderr);
    fflush(stderr);
    abort();
}

_Noreturn void fuse_rt_exit(int32_t status) {
    fflush(stdout);
    fflush(stderr);
    exit((int)status);
}
