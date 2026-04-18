/*
 * test_memory.c — exercise fuse_rt_alloc / realloc / free.
 *
 * Smoke-test only: allocates, writes, reads back, frees. Aborts
 * on any divergence from expected behaviour; exits 0 on success.
 */

#include "fuse_rt.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void check(int cond, const char *msg) {
    if (!cond) {
        fprintf(stderr, "test_memory: FAIL %s\n", msg);
        exit(1);
    }
}

int main(void) {
    /* Plain alloc/free. */
    uint8_t *a = (uint8_t *)fuse_rt_alloc(128, 16);
    check(a != NULL, "alloc 128/16 returned NULL");
    for (int i = 0; i < 128; i++) { a[i] = (uint8_t)i; }
    for (int i = 0; i < 128; i++) { check(a[i] == (uint8_t)i, "byte mismatch"); }

    /* Realloc up — old bytes preserved. */
    a = (uint8_t *)fuse_rt_realloc(a, 128, 256, 16);
    check(a != NULL, "realloc up returned NULL");
    for (int i = 0; i < 128; i++) { check(a[i] == (uint8_t)i, "byte mismatch post-realloc"); }

    /* Realloc down — still valid. */
    a = (uint8_t *)fuse_rt_realloc(a, 256, 64, 16);
    check(a != NULL, "realloc down returned NULL");
    for (int i = 0; i < 64; i++) { check(a[i] == (uint8_t)i, "byte mismatch post-shrink"); }

    fuse_rt_free(a, 64, 16);

    /* Zero-byte alloc returns NULL. */
    check(fuse_rt_alloc(0, 16) == NULL, "zero-size alloc should be NULL");

    /* Free on NULL is a no-op. */
    fuse_rt_free(NULL, 0, 16);

    printf("test_memory: ok\n");
    return 0;
}
