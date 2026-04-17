/*
 * abort.c — fuse_rt_abort implementation.
 *
 * At W05 the runtime has exactly one meaningful function: a clean
 * abort that prints a message to stderr and calls the host abort().
 * Everything else declared in fuse_rt.h is a stub that calls into
 * this to surface "not yet implemented" at runtime.
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
    /* Panic is a user-surface abort; the W11 work distinguishes it
       from fuse_rt_abort by formatting. At W05 both take the same
       path. */
    fuse_rt_abort(message == NULL ? "panic" : message);
}

/* Stubs for the surfaces W07/W16/W22 will flesh out. Each prints a
   diagnostic and aborts so a rogue codegen can't silently produce
   wrong output. */

int64_t fuse_rt_write_stdout(const uint8_t *bytes, int64_t len) {
    (void)bytes; (void)len;
    fuse_rt_abort("fuse_rt_write_stdout: not yet implemented (W22)");
}

int64_t fuse_rt_write_stderr(const uint8_t *bytes, int64_t len) {
    (void)bytes; (void)len;
    fuse_rt_abort("fuse_rt_write_stderr: not yet implemented (W22)");
}

void *fuse_rt_thread_spawn(void *(*entry)(void *), void *arg) {
    (void)entry; (void)arg;
    fuse_rt_abort("fuse_rt_thread_spawn: not yet implemented (W07/W16)");
}

int64_t fuse_rt_thread_join(void *handle) {
    (void)handle;
    fuse_rt_abort("fuse_rt_thread_join: not yet implemented (W07/W16)");
}

void *fuse_rt_chan_new(int64_t capacity) {
    (void)capacity;
    fuse_rt_abort("fuse_rt_chan_new: not yet implemented (W07/W16)");
}

int64_t fuse_rt_chan_send(void *chan, const void *value, int64_t bytes) {
    (void)chan; (void)value; (void)bytes;
    fuse_rt_abort("fuse_rt_chan_send: not yet implemented (W07/W16)");
}

int64_t fuse_rt_chan_recv(void *chan, void *out, int64_t bytes) {
    (void)chan; (void)out; (void)bytes;
    fuse_rt_abort("fuse_rt_chan_recv: not yet implemented (W07/W16)");
}
