/*
 * fuse_rt.h — Fuse runtime ABI surface.
 *
 * This header declares the ABI the Fuse codegen emits calls against.
 * At W05 every function is declared but only `fuse_rt_abort` has a
 * concrete implementation; the rest are stubs that the later waves
 * (W07 concurrency, W16 runtime ABI, W22 stdlib hosted) flesh out.
 *
 * A program that calls a stub runtime function gets a clean abort with
 * a message naming the unimplemented function; there is no silent
 * misbehavior (Rule 6.9).
 *
 * ABI stability: once a function is declared here, its signature is
 * fixed. Later waves may only add new functions; they may not change
 * existing ones (Rule 3.1 — the ABI boundary between compiler and
 * runtime is architecture, not plumbing).
 */
#ifndef FUSE_RT_H
#define FUSE_RT_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* -- Process control ---------------------------------------------------- */

/*
 * fuse_rt_abort terminates the process after writing `message` to
 * stderr. Used by runtime stubs to surface "this function is not yet
 * implemented" at program runtime rather than silent failure.
 *
 * Implemented in W05 (runtime/src/abort.c). Later waves harden the
 * message formatting and panic-hook integration.
 */
_Noreturn void fuse_rt_abort(const char *message);

/*
 * fuse_rt_panic is the user-facing panic primitive. W05 does not
 * wire it to real codegen; it is declared here so W11 error
 * propagation can name it in diagnostics.
 */
_Noreturn void fuse_rt_panic(const char *message);

/* -- IO ---------------------------------------------------------------- */

/*
 * fuse_rt_write_stdout writes `len` bytes from `bytes` to stdout.
 * W05 does not emit calls to this yet; it is declared here so the
 * W22 stdlib-hosted crate has a fixed signature to bind to.
 */
int64_t fuse_rt_write_stdout(const uint8_t *bytes, int64_t len);

/*
 * fuse_rt_write_stderr — same as above for stderr.
 */
int64_t fuse_rt_write_stderr(const uint8_t *bytes, int64_t len);

/* -- Concurrency stubs for W07 ----------------------------------------- */

/*
 * fuse_rt_thread_spawn / fuse_rt_thread_join cover the minimal
 * thread surface W07 will integrate. At W05 they are declared with
 * `void *` closure pointers so the C codegen contract is fixed.
 */
void *fuse_rt_thread_spawn(void *(*entry)(void *), void *arg);
int64_t fuse_rt_thread_join(void *handle);

/*
 * fuse_rt_chan_new / send / recv are the channel primitives W07
 * wires to HIR's KindChannel.
 */
void *fuse_rt_chan_new(int64_t capacity);
int64_t fuse_rt_chan_send(void *chan, const void *value, int64_t bytes);
int64_t fuse_rt_chan_recv(void *chan, void *out, int64_t bytes);

#ifdef __cplusplus
}
#endif

#endif /* FUSE_RT_H */
