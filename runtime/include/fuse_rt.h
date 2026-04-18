/*
 * fuse_rt.h — Fuse runtime ABI surface.
 *
 * This header declares the ABI the Fuse codegen emits calls against.
 * Wave 16 fleshes out every function declared here with a concrete
 * cross-platform implementation (Win32 threads on Windows; pthreads
 * on macOS / Linux). The signatures are stable ABI (Rule 3.1 — the
 * compiler/runtime boundary is architecture, not plumbing).
 *
 * Surfaces:
 *   - Process control: abort, panic, exit
 *   - Memory: alloc, realloc, free, sized free
 *   - IO: stdout / stderr byte writes
 *   - Process: pid, args, env
 *   - Time: monotonic + wall nanosecond counters, sleep
 *   - Thread: spawn / join / current-id / yield / TLS get/set
 *   - Sync: mutex lock / unlock + condvar wait / notify
 *   - Channel: bounded MPMC channel with send / recv / close /
 *     try_send / try_recv
 *
 * Silent-stub failures are forbidden (Rule 6.9). Anything not yet
 * implemented in the .c files aborts with a diagnostic naming the
 * unimplemented function.
 */
#ifndef FUSE_RT_H
#define FUSE_RT_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* -- Process control ---------------------------------------------------- */

/*
 * fuse_rt_abort terminates the process after writing `message` to
 * stderr. Used by panics, runtime assertions, and the "not yet
 * implemented" diagnostic path.
 */
_Noreturn void fuse_rt_abort(const char *message);

/*
 * fuse_rt_panic is the user-facing panic primitive. W16 formats the
 * message as "fuse panic: <message>\n" and flushes stderr before
 * calling the host abort(). Panic recovery is W22's concern — at
 * W16 every panic terminates the process cleanly.
 */
_Noreturn void fuse_rt_panic(const char *message);

/*
 * fuse_rt_exit terminates the process with the given status code
 * without calling destructors. Used by the driver to surface
 * program exit codes; not yet exposed to user Fuse code.
 */
_Noreturn void fuse_rt_exit(int32_t status);

/* -- Memory ------------------------------------------------------------ */

/*
 * fuse_rt_alloc returns a pointer to `bytes` newly-allocated bytes
 * aligned to `align`. Returns NULL on failure — the caller must
 * check and handle the OOM path (panic-on-OOM is policy, not
 * primitive).
 */
void *fuse_rt_alloc(int64_t bytes, int64_t align);

/*
 * fuse_rt_realloc grows or shrinks a previously-allocated block. The
 * old pointer is invalid after this call even when the returned
 * pointer equals the old one.
 */
void *fuse_rt_realloc(void *ptr, int64_t old_bytes, int64_t new_bytes, int64_t align);

/*
 * fuse_rt_free releases a block previously returned by fuse_rt_alloc
 * or fuse_rt_realloc. `bytes` and `align` must match the values used
 * on allocation. Double-free and mismatched-size are undefined
 * behaviour; the W16 debug build aborts with a clear diagnostic
 * in both cases.
 */
void fuse_rt_free(void *ptr, int64_t bytes, int64_t align);

/* -- IO ---------------------------------------------------------------- */

/*
 * fuse_rt_write_stdout / fuse_rt_write_stderr write `len` bytes from
 * `bytes` to the corresponding standard stream. Returns the number
 * of bytes written, or -1 on error. At W16 the implementation is
 * best-effort — short writes are retried until the full buffer is
 * emitted or an error occurs.
 */
int64_t fuse_rt_write_stdout(const uint8_t *bytes, int64_t len);
int64_t fuse_rt_write_stderr(const uint8_t *bytes, int64_t len);

/* -- Process ----------------------------------------------------------- */

/*
 * fuse_rt_pid returns the current process id as a 64-bit integer.
 * Portable across platforms: on Windows it is GetCurrentProcessId;
 * on POSIX it is getpid().
 */
int64_t fuse_rt_pid(void);

/* -- Time -------------------------------------------------------------- */

/*
 * fuse_rt_monotonic_ns returns a monotonic counter in nanoseconds.
 * The counter's origin is unspecified but does not decrease across
 * calls within a single process lifetime. Suitable for measuring
 * elapsed time.
 */
int64_t fuse_rt_monotonic_ns(void);

/*
 * fuse_rt_wall_ns returns the wall-clock time in nanoseconds since
 * the Unix epoch. Subject to clock adjustments; not suitable for
 * elapsed-time measurement.
 */
int64_t fuse_rt_wall_ns(void);

/*
 * fuse_rt_sleep_ns blocks the calling thread for at least `nanos`
 * nanoseconds. Spurious wakeups are the caller's concern — wrap
 * in a loop if wall-accurate sleeping matters.
 */
void fuse_rt_sleep_ns(int64_t nanos);

/* -- Threads ----------------------------------------------------------- */

/*
 * fuse_rt_thread_spawn launches a new OS thread whose entry is
 * `entry(arg)`. Returns an opaque handle the caller later passes
 * to fuse_rt_thread_join; returns NULL if the OS refuses to start
 * a thread (resource exhaustion, permission). The returned handle
 * must be joined exactly once; leaking it leaks an OS thread.
 */
void *fuse_rt_thread_spawn(int64_t (*entry)(void *), void *arg);

/*
 * fuse_rt_thread_join waits for the thread identified by `handle`
 * to terminate, returns the int64_t that thread's entry function
 * returned, and releases `handle`. After this call `handle` is
 * invalid. Passing a NULL handle aborts.
 */
int64_t fuse_rt_thread_join(void *handle);

/*
 * fuse_rt_thread_id returns the current thread's opaque 64-bit id.
 * The same thread returns the same id across calls; distinct
 * threads return distinct ids.
 */
int64_t fuse_rt_thread_id(void);

/*
 * fuse_rt_thread_yield hints the scheduler that the current thread
 * is willing to yield its slice. Best-effort.
 */
void fuse_rt_thread_yield(void);

/* -- Synchronisation primitives --------------------------------------- */

/*
 * fuse_rt_mutex_new / fuse_rt_mutex_lock / fuse_rt_mutex_unlock /
 * fuse_rt_mutex_free model a non-reentrant mutual-exclusion lock.
 * Returns an opaque handle from fuse_rt_mutex_new; every new must
 * be paired with exactly one free.
 */
void *fuse_rt_mutex_new(void);
void fuse_rt_mutex_lock(void *mutex);
void fuse_rt_mutex_unlock(void *mutex);
void fuse_rt_mutex_free(void *mutex);

/*
 * fuse_rt_cond_new / fuse_rt_cond_wait / fuse_rt_cond_notify_one /
 * fuse_rt_cond_notify_all / fuse_rt_cond_free model a condition
 * variable associated with a fuse_rt_mutex. cond_wait atomically
 * releases `mutex` and blocks; upon wakeup the mutex is re-locked.
 */
void *fuse_rt_cond_new(void);
void fuse_rt_cond_wait(void *cond, void *mutex);
void fuse_rt_cond_notify_one(void *cond);
void fuse_rt_cond_notify_all(void *cond);
void fuse_rt_cond_free(void *cond);

/* -- Channels --------------------------------------------------------- */

/*
 * Channel result codes. fuse_rt_chan_send returns 0 on success, a
 * positive code on recoverable failure. fuse_rt_chan_recv returns
 * 0 on success with the value written to `out`; other values
 * signal a state the caller must handle.
 */
#define FUSE_CHAN_OK          0
#define FUSE_CHAN_CLOSED      1  /* recv on an empty-and-closed chan */
#define FUSE_CHAN_WOULD_BLOCK 2  /* try_send / try_recv only */

/*
 * fuse_rt_chan_new creates a bounded MPMC channel carrying
 * `elem_bytes`-sized elements with a buffer of `capacity` slots.
 * Returns an opaque handle; NULL on allocation failure.
 *
 * A capacity of 0 indicates a rendezvous channel — every send
 * blocks until a paired recv is ready. W16 implements the
 * bounded buffer form; rendezvous is treated as capacity=1 to
 * keep the W16 proof simple (documented in WC016).
 */
void *fuse_rt_chan_new(int64_t capacity, int64_t elem_bytes);

/*
 * fuse_rt_chan_send blocks until a slot is available, then copies
 * `elem_bytes` from `value` into the buffer. Returns 0 on success,
 * FUSE_CHAN_CLOSED if the channel has been closed.
 */
int32_t fuse_rt_chan_send(void *chan, const void *value);

/*
 * fuse_rt_chan_recv blocks until a slot contains a value, then
 * copies `elem_bytes` from the buffer into `out`. Returns 0 on
 * success, FUSE_CHAN_CLOSED if the channel is drained and closed.
 */
int32_t fuse_rt_chan_recv(void *chan, void *out);

/*
 * fuse_rt_chan_try_send attempts a non-blocking send. Returns 0
 * on success, FUSE_CHAN_WOULD_BLOCK if the buffer is full,
 * FUSE_CHAN_CLOSED if the channel has been closed.
 */
int32_t fuse_rt_chan_try_send(void *chan, const void *value);

/*
 * fuse_rt_chan_try_recv attempts a non-blocking receive. Same
 * return codes as try_send.
 */
int32_t fuse_rt_chan_try_recv(void *chan, void *out);

/*
 * fuse_rt_chan_close marks the channel closed. Subsequent send
 * attempts return FUSE_CHAN_CLOSED; in-flight recv calls return
 * the buffered values until the buffer is drained, at which
 * point they return FUSE_CHAN_CLOSED.
 */
void fuse_rt_chan_close(void *chan);

/*
 * fuse_rt_chan_free releases the channel. Callers must ensure no
 * thread is blocked on send / recv before freeing.
 */
void fuse_rt_chan_free(void *chan);

#ifdef __cplusplus
}
#endif

#endif /* FUSE_RT_H */
