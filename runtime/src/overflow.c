/*
 * overflow.c — cross-compiler signed 64-bit overflow primitives.
 *
 * The Fuse codegen emits calls to fuse_rt_{add,sub,mul}_overflow_i64
 * for overflow-policy arithmetic (reference §33.1). On GCC and Clang
 * hosts the compiler-builtin path is the fast route; on MSVC — which
 * does not ship the __builtin_*_overflow intrinsics — a portable
 * pure-C branch-and-compare preserves semantics without linker
 * failure. W24 retires the "MSVC overflow fallback pending" STUBS
 * row by landing this file.
 *
 * Each function returns 1 when the mathematical result overflows
 * the INT64 range, else 0. Even on overflow the wrapped result is
 * written through `out`, matching the GCC / Clang builtin contract.
 */

#include "fuse_rt.h"

#include <stdint.h>

#if defined(__GNUC__) || defined(__clang__)

int32_t fuse_rt_add_overflow_i64(int64_t a, int64_t b, int64_t *out) {
    return __builtin_add_overflow(a, b, out) ? 1 : 0;
}

int32_t fuse_rt_sub_overflow_i64(int64_t a, int64_t b, int64_t *out) {
    return __builtin_sub_overflow(a, b, out) ? 1 : 0;
}

int32_t fuse_rt_mul_overflow_i64(int64_t a, int64_t b, int64_t *out) {
    return __builtin_mul_overflow(a, b, out) ? 1 : 0;
}

#else

/*
 * Portable fallback. Computes the wrapped result through uint64
 * arithmetic (two's-complement wrap is defined on unsigned) and
 * checks for overflow via the signed-operand comparison described
 * in Hacker's Delight §2-13.
 */
int32_t fuse_rt_add_overflow_i64(int64_t a, int64_t b, int64_t *out) {
    uint64_t ua = (uint64_t)a;
    uint64_t ub = (uint64_t)b;
    uint64_t ur = ua + ub;
    *out = (int64_t)ur;
    /* Overflow iff (a^r) & (b^r) has the sign bit set. */
    return (int32_t)(((ua ^ ur) & (ub ^ ur)) >> 63);
}

int32_t fuse_rt_sub_overflow_i64(int64_t a, int64_t b, int64_t *out) {
    uint64_t ua = (uint64_t)a;
    uint64_t ub = (uint64_t)b;
    uint64_t ur = ua - ub;
    *out = (int64_t)ur;
    /* Overflow iff (a^b) & (a^r) has the sign bit set. */
    return (int32_t)(((ua ^ ub) & (ua ^ ur)) >> 63);
}

int32_t fuse_rt_mul_overflow_i64(int64_t a, int64_t b, int64_t *out) {
    /* Split and re-combine so we can detect overflow before it lands
     * in the destination. The fast path uses a wider native integer
     * when available; __int128 is GCC / Clang — so on the MSVC
     * branch we do the explicit signed-range test below.
     */
    if (a == 0 || b == 0) {
        *out = 0;
        return 0;
    }
    int64_t r = (int64_t)((uint64_t)a * (uint64_t)b);
    *out = r;
    /* a * b overflowed iff r / a != b (guarding INT64_MIN * -1 case
     * which is undefined as signed division). */
    if (a == INT64_MIN && b == -1) return 1;
    if (b == INT64_MIN && a == -1) return 1;
    if ((r / a) != b) return 1;
    return 0;
}

#endif
