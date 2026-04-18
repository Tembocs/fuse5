/*
 * memory.c — allocation primitives.
 *
 * W16 uses the host C library's aligned allocator. A later wave
 * (W21 custom allocators) will lift this to a pluggable
 * Allocator trait; until then the system malloc is the one and
 * only source.
 *
 * Portability: `aligned_alloc` landed in C11 and is available on
 * modern glibc / macOS. Windows does not provide aligned_alloc;
 * we fall back to `_aligned_malloc` / `_aligned_free`.
 */

#include "fuse_rt.h"

#include <stdlib.h>
#include <string.h>

#if defined(_WIN32)
#include <malloc.h> /* _aligned_malloc, _aligned_free */
#endif

#if !defined(_WIN32)
/* Round `bytes` up to the next multiple of `align`. aligned_alloc
   requires that the size be a multiple of align on glibc, so we
   pad every allocation upward. Windows' _aligned_malloc has no
   such requirement, so the helper is POSIX-only. */
static int64_t round_up(int64_t bytes, int64_t align) {
    if (align <= 1) {
        return bytes;
    }
    int64_t rem = bytes % align;
    if (rem == 0) {
        return bytes;
    }
    return bytes + (align - rem);
}
#endif

void *fuse_rt_alloc(int64_t bytes, int64_t align) {
    if (bytes <= 0) {
        return NULL;
    }
    if (align <= 0) {
        align = 1;
    }
#if defined(_WIN32)
    return _aligned_malloc((size_t)bytes, (size_t)align);
#else
    size_t padded = (size_t)round_up(bytes, align);
    return aligned_alloc((size_t)align, padded);
#endif
}

void *fuse_rt_realloc(void *ptr, int64_t old_bytes, int64_t new_bytes, int64_t align) {
    (void)old_bytes;
    if (new_bytes <= 0) {
        fuse_rt_free(ptr, old_bytes, align);
        return NULL;
    }
    if (ptr == NULL) {
        return fuse_rt_alloc(new_bytes, align);
    }
#if defined(_WIN32)
    return _aligned_realloc(ptr, (size_t)new_bytes, (size_t)align);
#else
    /* POSIX has no aligned realloc. Allocate fresh, copy, free. */
    void *fresh = fuse_rt_alloc(new_bytes, align);
    if (fresh == NULL) {
        return NULL;
    }
    int64_t copy_bytes = old_bytes < new_bytes ? old_bytes : new_bytes;
    if (copy_bytes > 0) {
        memcpy(fresh, ptr, (size_t)copy_bytes);
    }
    fuse_rt_free(ptr, old_bytes, align);
    return fresh;
#endif
}

void fuse_rt_free(void *ptr, int64_t bytes, int64_t align) {
    (void)bytes;
    (void)align;
    if (ptr == NULL) {
        return;
    }
#if defined(_WIN32)
    _aligned_free(ptr);
#else
    free(ptr);
#endif
}
