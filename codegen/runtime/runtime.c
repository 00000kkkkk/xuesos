/*
 * =============================================================================
 *  Xuesos++ Runtime Library
 * =============================================================================
 *
 *  This file provides the complete C runtime support for compiled Xuesos++
 *  programs. The code generator (codegen.go) emits C code that links against
 *  these functions and types at compile time.
 *
 *  Compile with:
 *      gcc -c runtime.c -lpthread -lm
 *
 *  Sections:
 *      1. Includes and Forward Declarations
 *      2. Memory Management (Simple Mark-and-Sweep GC)
 *      3. Strings (Heap-allocated, GC-managed)
 *      4. Dynamic Arrays
 *      5. Hash Maps
 *      6. Print / IO Functions
 *      7. Type Conversion Helpers
 *      8. Error Handling (xutry / xucatch / xuthrow)
 *      9. Concurrency (pthreads: xuspawn, channels)
 *     10. Defer Stack
 *     11. Math Helpers
 *     12. Cleanup
 *
 * =============================================================================
 */

/* ============================================================================
 * 1. Includes and Forward Declarations
 * ============================================================================ */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>
#include <math.h>
#include <stdarg.h>

#ifdef _WIN32
    #include <windows.h>
    /* On Windows, we use Win32 threads instead of pthreads unless
       the user has a pthreads-win32 library. We provide a thin
       abstraction layer. */
    typedef HANDLE            xpp_thread_t;
    typedef CRITICAL_SECTION  xpp_mutex_t;
    typedef CONDITION_VARIABLE xpp_cond_t;

    static void xpp_mutex_init(xpp_mutex_t *m)    { InitializeCriticalSection(m); }
    static void xpp_mutex_lock(xpp_mutex_t *m)    { EnterCriticalSection(m); }
    static void xpp_mutex_unlock(xpp_mutex_t *m)  { LeaveCriticalSection(m); }
    static void xpp_mutex_destroy(xpp_mutex_t *m) { DeleteCriticalSection(m); }

    static void xpp_cond_init(xpp_cond_t *c)                    { InitializeConditionVariable(c); }
    static void xpp_cond_wait(xpp_cond_t *c, xpp_mutex_t *m)    { SleepConditionVariableCS(c, m, INFINITE); }
    static void xpp_cond_signal(xpp_cond_t *c)                  { WakeConditionVariable(c); }
    static void xpp_cond_destroy(xpp_cond_t *c)                 { (void)c; /* no-op on Windows */ }
#else
    #include <pthread.h>
    typedef pthread_t         xpp_thread_t;
    typedef pthread_mutex_t   xpp_mutex_t;
    typedef pthread_cond_t    xpp_cond_t;

    static void xpp_mutex_init(xpp_mutex_t *m)    { pthread_mutex_init(m, NULL); }
    static void xpp_mutex_lock(xpp_mutex_t *m)    { pthread_mutex_lock(m); }
    static void xpp_mutex_unlock(xpp_mutex_t *m)  { pthread_mutex_unlock(m); }
    static void xpp_mutex_destroy(xpp_mutex_t *m) { pthread_mutex_destroy(m); }

    static void xpp_cond_init(xpp_cond_t *c)                    { pthread_cond_init(c, NULL); }
    static void xpp_cond_wait(xpp_cond_t *c, xpp_mutex_t *m)    { pthread_cond_wait(c, m); }
    static void xpp_cond_signal(xpp_cond_t *c)                  { pthread_cond_signal(c); }
    static void xpp_cond_destroy(xpp_cond_t *c)                 { pthread_cond_destroy(c); }
#endif


/* ============================================================================
 * 2. Memory Management  --  Simple Mark-and-Sweep Garbage Collector
 * ============================================================================
 *
 * Every heap object allocated through the runtime carries an XppObj header.
 * All objects are linked into a global intrusive list so the GC can walk them.
 *
 * The current implementation is conservative: when the object count exceeds a
 * threshold we simply double the threshold (a real GC would scan roots and
 * sweep unreachable objects). This keeps the runtime simple while still giving
 * us a single point where a proper collector can be plugged in later.
 */

/* Object type tags */
#define XPP_OBJ_STRING  1
#define XPP_OBJ_ARRAY   2
#define XPP_OBJ_MAP     3

/* Common header prepended to every GC-managed object */
typedef struct XppObj {
    int  type;           /* XPP_OBJ_STRING, XPP_OBJ_ARRAY, ... */
    int  marked;         /* used by mark phase                   */
    struct XppObj *next; /* intrusive linked list of all objects  */
} XppObj;

/* ---- Global GC state ---- */
static XppObj *gc_objects    = NULL;
static int     gc_count      = 0;
static int     gc_threshold  = 256;

/*
 * xpp_alloc -- allocate a GC-tracked object of the given size and type tag.
 *
 * When the object count reaches the threshold we grow the threshold.
 * In a production runtime this is where mark-and-sweep would run.
 */
static void *xpp_alloc(size_t size, int type) {
    if (gc_count >= gc_threshold) {
        /* TODO: implement mark phase (scan stack roots) and sweep phase.
           For now we just grow the threshold so the program keeps running. */
        gc_threshold *= 2;
    }

    XppObj *obj = (XppObj *)malloc(size);
    if (!obj) {
        fprintf(stderr, "runtime error: out of memory (requested %zu bytes)\n", size);
        exit(1);
    }
    obj->type   = type;
    obj->marked = 0;
    obj->next   = gc_objects;
    gc_objects   = obj;
    gc_count++;
    return obj;
}

/*
 * xpp_gc_free_all -- release every object on the GC list.
 * Called once at program exit via the destructor / atexit.
 */
static void xpp_gc_free_all(void) {
    XppObj *obj = gc_objects;
    while (obj) {
        XppObj *next = obj->next;

        /* Free type-specific inner allocations before the object itself */
        switch (obj->type) {
        case XPP_OBJ_STRING: {
            /* XppString stores a separate malloc'd char buffer */
            char **data_ptr = (char **)((char *)obj + sizeof(XppObj));
            if (*data_ptr) free(*data_ptr);
            break;
        }
        case XPP_OBJ_ARRAY: {
            /* XppArray stores a malloc'd void** buffer */
            void ***data_ptr = (void ***)((char *)obj + sizeof(XppObj));
            if (*data_ptr) free(*data_ptr);
            break;
        }
        case XPP_OBJ_MAP: {
            /* XppMap stores a malloc'd bucket array; entries are chained */
            /* We handle this below in a dedicated helper */
            break;
        }
        default:
            break;
        }

        free(obj);
        obj = next;
    }
    gc_objects = NULL;
    gc_count   = 0;
}


/* ============================================================================
 * 3. Strings  --  Heap-allocated, GC-managed
 * ============================================================================
 *
 * XppString is the runtime representation of the Xuesos++ `str` type.
 * Each string owns a null-terminated char buffer.
 */

typedef struct {
    XppObj   obj;     /* GC header -- must be first */
    char    *data;    /* null-terminated UTF-8 bytes */
    int64_t  len;     /* length in bytes (excluding NUL) */
} XppString;

/* Create a new XppString from a C string literal */
static XppString *xpp_string_new(const char *s) {
    XppString *str = (XppString *)xpp_alloc(sizeof(XppString), XPP_OBJ_STRING);
    str->len  = (int64_t)strlen(s);
    str->data = (char *)malloc((size_t)str->len + 1);
    if (!str->data) {
        fprintf(stderr, "runtime error: out of memory allocating string\n");
        exit(1);
    }
    memcpy(str->data, s, (size_t)str->len + 1);
    return str;
}

/* Concatenate two strings, returning a new string */
static XppString *xpp_string_concat(XppString *a, XppString *b) {
    XppString *result = (XppString *)xpp_alloc(sizeof(XppString), XPP_OBJ_STRING);
    result->len  = a->len + b->len;
    result->data = (char *)malloc((size_t)result->len + 1);
    if (!result->data) {
        fprintf(stderr, "runtime error: out of memory concatenating strings\n");
        exit(1);
    }
    memcpy(result->data, a->data, (size_t)a->len);
    memcpy(result->data + a->len, b->data, (size_t)b->len + 1);
    return result;
}

/* Equality check (byte-for-byte) */
static bool xpp_string_eq(XppString *a, XppString *b) {
    if (a == b) return true;
    if (a->len != b->len) return false;
    return memcmp(a->data, b->data, (size_t)a->len) == 0;
}

/* Inequality check */
static bool xpp_string_neq(XppString *a, XppString *b) {
    return !xpp_string_eq(a, b);
}

/* Length in bytes */
static int64_t xpp_string_len(XppString *s) {
    return s->len;
}

/* Substring [start, end) -- returns new string */
static XppString *xpp_string_slice(XppString *s, int64_t start, int64_t end) {
    if (start < 0) start = 0;
    if (end > s->len) end = s->len;
    if (start >= end) return xpp_string_new("");

    int64_t slice_len = end - start;
    XppString *result = (XppString *)xpp_alloc(sizeof(XppString), XPP_OBJ_STRING);
    result->len  = slice_len;
    result->data = (char *)malloc((size_t)slice_len + 1);
    memcpy(result->data, s->data + start, (size_t)slice_len);
    result->data[slice_len] = '\0';
    return result;
}

/* Index into string -- returns the byte at position idx as a char */
static char xpp_string_char_at(XppString *s, int64_t idx) {
    if (idx < 0 || idx >= s->len) {
        fprintf(stderr, "runtime error: string index %lld out of bounds (len=%lld)\n",
                (long long)idx, (long long)s->len);
        exit(1);
    }
    return s->data[idx];
}

/* Find substring, returns index or -1 */
static int64_t xpp_string_find(XppString *haystack, XppString *needle) {
    if (needle->len == 0) return 0;
    if (needle->len > haystack->len) return -1;
    char *p = strstr(haystack->data, needle->data);
    if (!p) return -1;
    return (int64_t)(p - haystack->data);
}

/* Check if string contains a substring */
static bool xpp_string_contains(XppString *haystack, XppString *needle) {
    return xpp_string_find(haystack, needle) >= 0;
}

/* Convert various types to XppString */
static XppString *xpp_string_from_int(int64_t n) {
    char buf[32];
    snprintf(buf, sizeof(buf), "%lld", (long long)n);
    return xpp_string_new(buf);
}

static XppString *xpp_string_from_float(double n) {
    char buf[64];
    snprintf(buf, sizeof(buf), "%g", n);
    return xpp_string_new(buf);
}

static XppString *xpp_string_from_bool(bool b) {
    return xpp_string_new(b ? "xuitru" : "xuinia");
}

static XppString *xpp_string_from_char(char c) {
    char buf[2] = { c, '\0' };
    return xpp_string_new(buf);
}

/*
 * xpp_string_format -- sprintf-style formatting into a new XppString.
 * Useful for string interpolation in generated code.
 */
static XppString *xpp_string_format(const char *fmt, ...) {
    va_list args1, args2;
    va_start(args1, fmt);
    va_copy(args2, args1);

    int needed = vsnprintf(NULL, 0, fmt, args1);
    va_end(args1);

    char *buf = (char *)malloc((size_t)needed + 1);
    if (!buf) {
        fprintf(stderr, "runtime error: out of memory formatting string\n");
        exit(1);
    }
    vsnprintf(buf, (size_t)needed + 1, fmt, args2);
    va_end(args2);

    XppString *result = xpp_string_new(buf);
    free(buf);
    return result;
}

/*
 * C-string comparison helper used by the code generator for simple
 * string comparisons (e.g., in xuiatch arms with string patterns).
 */
static int xpp_streq(const char *a, const char *b) {
    return strcmp(a, b) == 0;
}


/* ============================================================================
 * 4. Dynamic Arrays
 * ============================================================================
 *
 * XppArray is the runtime representation of Xuesos++ array/slice types.
 * It stores a flat array of void* pointers and grows geometrically.
 */

typedef struct {
    XppObj   obj;   /* GC header -- must be first */
    void   **data;
    int64_t  len;
    int64_t  cap;
} XppArray;

/* Create a new empty array with the given initial capacity */
static XppArray *xpp_array_new(int64_t cap) {
    XppArray *arr = (XppArray *)xpp_alloc(sizeof(XppArray), XPP_OBJ_ARRAY);
    arr->len  = 0;
    arr->cap  = cap > 0 ? cap : 8;
    arr->data = (void **)malloc(sizeof(void *) * (size_t)arr->cap);
    if (!arr->data) {
        fprintf(stderr, "runtime error: out of memory allocating array\n");
        exit(1);
    }
    return arr;
}

/* Push a value onto the end of the array */
static void xpp_array_push(XppArray *arr, void *val) {
    if (arr->len >= arr->cap) {
        arr->cap *= 2;
        arr->data = (void **)realloc(arr->data, sizeof(void *) * (size_t)arr->cap);
        if (!arr->data) {
            fprintf(stderr, "runtime error: out of memory growing array\n");
            exit(1);
        }
    }
    arr->data[arr->len++] = val;
}

/* Pop the last element and return it */
static void *xpp_array_pop(XppArray *arr) {
    if (arr->len == 0) {
        fprintf(stderr, "runtime error: pop from empty array\n");
        exit(1);
    }
    return arr->data[--arr->len];
}

/* Bounds-checked get */
static void *xpp_array_get(XppArray *arr, int64_t idx) {
    if (idx < 0 || idx >= arr->len) {
        fprintf(stderr, "runtime error: array index %lld out of bounds (len=%lld)\n",
                (long long)idx, (long long)arr->len);
        exit(1);
    }
    return arr->data[idx];
}

/* Bounds-checked set */
static void xpp_array_set(XppArray *arr, int64_t idx, void *val) {
    if (idx < 0 || idx >= arr->len) {
        fprintf(stderr, "runtime error: array index %lld out of bounds (len=%lld)\n",
                (long long)idx, (long long)arr->len);
        exit(1);
    }
    arr->data[idx] = val;
}

/* Return current length */
static int64_t xpp_array_len(XppArray *arr) {
    return arr->len;
}

/* Return current capacity */
static int64_t xpp_array_cap(XppArray *arr) {
    return arr->cap;
}

/* Slice an array [start, end) into a new array (shallow copy) */
static XppArray *xpp_array_slice(XppArray *arr, int64_t start, int64_t end) {
    if (start < 0) start = 0;
    if (end > arr->len) end = arr->len;
    if (start >= end) return xpp_array_new(0);

    int64_t new_len = end - start;
    XppArray *result = xpp_array_new(new_len);
    memcpy(result->data, arr->data + start, sizeof(void *) * (size_t)new_len);
    result->len = new_len;
    return result;
}


/* ============================================================================
 * 5. Hash Maps
 * ============================================================================
 *
 * XppMap is a string-keyed hash map using separate chaining.
 * It uses the djb2 hash function.
 */

#define XPP_MAP_INIT_CAP     16
#define XPP_MAP_LOAD_FACTOR  0.75

typedef struct XppMapEntry {
    char                *key;
    void                *value;
    struct XppMapEntry  *next;   /* chain for collisions */
} XppMapEntry;

typedef struct {
    XppObj         obj;       /* GC header -- must be first */
    XppMapEntry  **buckets;
    int64_t        cap;
    int64_t        len;
} XppMap;

/* djb2 hash */
static uint64_t xpp_hash(const char *key) {
    uint64_t hash = 5381;
    while (*key) {
        hash = ((hash << 5) + hash) + (unsigned char)*key++;
    }
    return hash;
}

/* Create a new empty map */
static XppMap *xpp_map_new(void) {
    XppMap *m = (XppMap *)xpp_alloc(sizeof(XppMap), XPP_OBJ_MAP);
    m->cap     = XPP_MAP_INIT_CAP;
    m->len     = 0;
    m->buckets = (XppMapEntry **)calloc((size_t)m->cap, sizeof(XppMapEntry *));
    if (!m->buckets) {
        fprintf(stderr, "runtime error: out of memory allocating map\n");
        exit(1);
    }
    return m;
}

/* Internal: rehash the map when load factor is exceeded */
static void xpp_map_rehash(XppMap *m) {
    int64_t old_cap = m->cap;
    XppMapEntry **old_buckets = m->buckets;

    m->cap *= 2;
    m->buckets = (XppMapEntry **)calloc((size_t)m->cap, sizeof(XppMapEntry *));
    if (!m->buckets) {
        fprintf(stderr, "runtime error: out of memory rehashing map\n");
        exit(1);
    }

    /* Re-insert all entries into the new bucket array */
    for (int64_t i = 0; i < old_cap; i++) {
        XppMapEntry *entry = old_buckets[i];
        while (entry) {
            XppMapEntry *next = entry->next;
            uint64_t idx = xpp_hash(entry->key) % (uint64_t)m->cap;
            entry->next = m->buckets[idx];
            m->buckets[idx] = entry;
            entry = next;
        }
    }

    free(old_buckets);
}

/* Set a key-value pair (insert or update) */
static void xpp_map_set(XppMap *m, const char *key, void *value) {
    uint64_t idx = xpp_hash(key) % (uint64_t)m->cap;
    XppMapEntry *entry = m->buckets[idx];

    /* Check if key already exists */
    while (entry) {
        if (strcmp(entry->key, key) == 0) {
            entry->value = value;
            return;
        }
        entry = entry->next;
    }

    /* Key not found -- insert new entry */
    XppMapEntry *new_entry = (XppMapEntry *)malloc(sizeof(XppMapEntry));
    if (!new_entry) {
        fprintf(stderr, "runtime error: out of memory inserting into map\n");
        exit(1);
    }
    new_entry->key   = strdup(key);
    new_entry->value = value;
    new_entry->next  = m->buckets[idx];
    m->buckets[idx]  = new_entry;
    m->len++;

    /* Rehash if load factor exceeded */
    if ((double)m->len / (double)m->cap > XPP_MAP_LOAD_FACTOR) {
        xpp_map_rehash(m);
    }
}

/* Get a value by key. Returns NULL if not found. */
static void *xpp_map_get(XppMap *m, const char *key) {
    uint64_t idx = xpp_hash(key) % (uint64_t)m->cap;
    XppMapEntry *entry = m->buckets[idx];
    while (entry) {
        if (strcmp(entry->key, key) == 0) return entry->value;
        entry = entry->next;
    }
    return NULL;
}

/* Check if a key exists */
static bool xpp_map_has(XppMap *m, const char *key) {
    return xpp_map_get(m, key) != NULL;
}

/* Delete a key-value pair. Returns true if the key was found. */
static bool xpp_map_delete(XppMap *m, const char *key) {
    uint64_t idx = xpp_hash(key) % (uint64_t)m->cap;
    XppMapEntry *entry = m->buckets[idx];
    XppMapEntry *prev  = NULL;

    while (entry) {
        if (strcmp(entry->key, key) == 0) {
            if (prev) {
                prev->next = entry->next;
            } else {
                m->buckets[idx] = entry->next;
            }
            free(entry->key);
            free(entry);
            m->len--;
            return true;
        }
        prev  = entry;
        entry = entry->next;
    }
    return false;
}

/* Return the number of entries */
static int64_t xpp_map_len(XppMap *m) {
    return m->len;
}


/* ============================================================================
 * 6. Print / IO Functions
 * ============================================================================
 *
 * These are called by generated code for the `print()` / `println()` builtins.
 * Boolean values print as the Xuesos++ keywords "xuitru" / "xuinia".
 */

static void xpp_print_int(int64_t val) {
    printf("%lld\n", (long long)val);
}

static void xpp_print_float(double val) {
    printf("%g\n", val);
}

static void xpp_print_string(XppString *val) {
    if (val && val->data) {
        printf("%.*s\n", (int)val->len, val->data);
    } else {
        printf("xuinull\n");
    }
}

static void xpp_print_bool(bool val) {
    printf("%s\n", val ? "xuitru" : "xuinia");
}

static void xpp_print_char(char val) {
    printf("%c\n", val);
}

static void xpp_print_cstr(const char *val) {
    if (val) {
        printf("%s\n", val);
    } else {
        printf("xuinull\n");
    }
}

/* Print without a trailing newline (for inline formatting) */
static void xpp_print_int_nonl(int64_t val)    { printf("%lld", (long long)val); }
static void xpp_print_float_nonl(double val)    { printf("%g", val); }
static void xpp_print_bool_nonl(bool val)       { printf("%s", val ? "xuitru" : "xuinia"); }
static void xpp_print_cstr_nonl(const char *val){ printf("%s", val ? val : "xuinull"); }

static void xpp_print_string_nonl(XppString *val) {
    if (val && val->data) {
        printf("%.*s", (int)val->len, val->data);
    } else {
        printf("xuinull");
    }
}

/* Read a line from stdin into a new XppString */
static XppString *xpp_read_line(void) {
    char buf[4096];
    if (!fgets(buf, sizeof(buf), stdin)) {
        return xpp_string_new("");
    }
    /* Strip trailing newline */
    size_t len = strlen(buf);
    if (len > 0 && buf[len - 1] == '\n') {
        buf[len - 1] = '\0';
        if (len > 1 && buf[len - 2] == '\r') {
            buf[len - 2] = '\0';
        }
    }
    return xpp_string_new(buf);
}


/* ============================================================================
 * 7. Type Conversion Helpers
 * ============================================================================ */

/* Parse a string to int64_t. Returns 0 on failure. */
static int64_t xpp_parse_int(XppString *s) {
    if (!s || !s->data) return 0;
    return (int64_t)strtoll(s->data, NULL, 10);
}

/* Parse a string to double. Returns 0.0 on failure. */
static double xpp_parse_float(XppString *s) {
    if (!s || !s->data) return 0.0;
    return strtod(s->data, NULL);
}

/* Cast int to float */
static double xpp_int_to_float(int64_t n) {
    return (double)n;
}

/* Cast float to int (truncating) */
static int64_t xpp_float_to_int(double n) {
    return (int64_t)n;
}

/* Box primitive values into void* for use in arrays/maps.
 * These allocate a small block on the heap for the value. */
static void *xpp_box_int(int64_t val) {
    int64_t *p = (int64_t *)malloc(sizeof(int64_t));
    *p = val;
    return (void *)p;
}

static void *xpp_box_float(double val) {
    double *p = (double *)malloc(sizeof(double));
    *p = val;
    return (void *)p;
}

static void *xpp_box_bool(bool val) {
    bool *p = (bool *)malloc(sizeof(bool));
    *p = val;
    return (void *)p;
}

/* Unbox primitives from void* */
static int64_t xpp_unbox_int(void *p)  { return *(int64_t *)p; }
static double  xpp_unbox_float(void *p){ return *(double *)p; }
static bool    xpp_unbox_bool(void *p) { return *(bool *)p; }


/* ============================================================================
 * 8. Error Handling  --  xutry / xucatch / xuthrow
 * ============================================================================
 *
 * Xuesos++ uses xutry/xucatch/xuthrow for error handling.
 * We implement this with a simple global error flag and message.
 * The code generator wraps xutry bodies and checks _xpp_has_error
 * after each statement to branch into the xucatch handler.
 *
 * For nested xutry blocks the generator saves and restores the error state.
 */

static int         _xpp_has_error  = 0;
static const char *_xpp_error_msg  = "";

/* Raise an error (called from generated code for `xuthrow`) */
static void xpp_throw(const char *msg) {
    _xpp_has_error = 1;
    _xpp_error_msg = msg;
}

/* Clear the error state (called at the start of a xutry block) */
static void xpp_error_clear(void) {
    _xpp_has_error = 0;
    _xpp_error_msg = "";
}

/* Check if an error is pending */
static bool xpp_has_error(void) {
    return _xpp_has_error != 0;
}

/* Get the current error message */
static const char *xpp_get_error(void) {
    return _xpp_error_msg;
}


/* ============================================================================
 * 9. Concurrency  --  xuspawn (pthreads / Win32 threads), Channels
 * ============================================================================
 *
 * `xuspawn` launches a new OS thread running the given function.
 * Channels provide synchronous (unbuffered) communication between threads.
 */

/* ---- xuspawn ---- */

typedef struct {
    void (*fn)(void *);
    void *arg;
} XppSpawnArg;

#ifdef _WIN32
static DWORD WINAPI xpp_spawn_wrapper_win32(LPVOID arg) {
    XppSpawnArg *sa = (XppSpawnArg *)arg;
    sa->fn(sa->arg);
    free(sa);
    return 0;
}

static void xpp_spawn(void (*fn)(void *), void *arg) {
    XppSpawnArg *sa = (XppSpawnArg *)malloc(sizeof(XppSpawnArg));
    if (!sa) {
        fprintf(stderr, "runtime error: out of memory spawning thread\n");
        exit(1);
    }
    sa->fn  = fn;
    sa->arg = arg;

    HANDLE thread = CreateThread(NULL, 0, xpp_spawn_wrapper_win32, sa, 0, NULL);
    if (thread) {
        CloseHandle(thread); /* detach */
    } else {
        fprintf(stderr, "runtime error: failed to spawn thread\n");
        free(sa);
        exit(1);
    }
}
#else
static void *xpp_spawn_wrapper_posix(void *arg) {
    XppSpawnArg *sa = (XppSpawnArg *)arg;
    sa->fn(sa->arg);
    free(sa);
    return NULL;
}

static void xpp_spawn(void (*fn)(void *), void *arg) {
    XppSpawnArg *sa = (XppSpawnArg *)malloc(sizeof(XppSpawnArg));
    if (!sa) {
        fprintf(stderr, "runtime error: out of memory spawning thread\n");
        exit(1);
    }
    sa->fn  = fn;
    sa->arg = arg;

    pthread_t thread;
    int err = pthread_create(&thread, NULL, xpp_spawn_wrapper_posix, sa);
    if (err != 0) {
        fprintf(stderr, "runtime error: failed to spawn thread (errno=%d)\n", err);
        free(sa);
        exit(1);
    }
    pthread_detach(thread);
}
#endif

/* ---- Channels (synchronous / unbuffered) ---- */

typedef struct {
    void       *value;
    bool        has_value;
    bool        closed;
    xpp_mutex_t mutex;
    xpp_cond_t  send_cond;
    xpp_cond_t  recv_cond;
} XppChannel;

/* Create a new unbuffered channel */
static XppChannel *xpp_channel_new(void) {
    XppChannel *ch = (XppChannel *)malloc(sizeof(XppChannel));
    if (!ch) {
        fprintf(stderr, "runtime error: out of memory creating channel\n");
        exit(1);
    }
    ch->has_value = false;
    ch->closed    = false;
    ch->value     = NULL;
    xpp_mutex_init(&ch->mutex);
    xpp_cond_init(&ch->send_cond);
    xpp_cond_init(&ch->recv_cond);
    return ch;
}

/* Send a value into the channel (blocks until a receiver picks it up) */
static void xpp_channel_send(XppChannel *ch, void *val) {
    xpp_mutex_lock(&ch->mutex);
    while (ch->has_value && !ch->closed) {
        xpp_cond_wait(&ch->send_cond, &ch->mutex);
    }
    if (ch->closed) {
        xpp_mutex_unlock(&ch->mutex);
        fprintf(stderr, "runtime error: send on closed channel\n");
        exit(1);
    }
    ch->value     = val;
    ch->has_value = true;
    xpp_cond_signal(&ch->recv_cond);
    xpp_mutex_unlock(&ch->mutex);
}

/* Receive a value from the channel (blocks until a sender provides one) */
static void *xpp_channel_recv(XppChannel *ch) {
    xpp_mutex_lock(&ch->mutex);
    while (!ch->has_value && !ch->closed) {
        xpp_cond_wait(&ch->recv_cond, &ch->mutex);
    }
    if (!ch->has_value && ch->closed) {
        xpp_mutex_unlock(&ch->mutex);
        return NULL; /* channel closed with no value */
    }
    void *val     = ch->value;
    ch->has_value = false;
    xpp_cond_signal(&ch->send_cond);
    xpp_mutex_unlock(&ch->mutex);
    return val;
}

/* Close a channel -- wakes any waiting senders/receivers */
static void xpp_channel_close(XppChannel *ch) {
    xpp_mutex_lock(&ch->mutex);
    ch->closed = true;
    xpp_cond_signal(&ch->send_cond);
    xpp_cond_signal(&ch->recv_cond);
    xpp_mutex_unlock(&ch->mutex);
}

/* Free a channel's resources */
static void xpp_channel_free(XppChannel *ch) {
    xpp_mutex_destroy(&ch->mutex);
    xpp_cond_destroy(&ch->send_cond);
    xpp_cond_destroy(&ch->recv_cond);
    free(ch);
}


/* ============================================================================
 * 10. Defer Stack
 * ============================================================================
 *
 * `xudefer` pushes a cleanup function onto a per-scope stack.
 * At scope exit the code generator emits calls to xpp_defer_run_all().
 * Each deferred action is a (function pointer, argument) pair.
 */

#define XPP_DEFER_MAX 256

typedef struct {
    void (*fn)(void *);
    void *arg;
} XppDeferEntry;

typedef struct {
    XppDeferEntry entries[XPP_DEFER_MAX];
    int           count;
} XppDeferStack;

/* Initialize a defer stack (typically one per function scope) */
static void xpp_defer_init(XppDeferStack *ds) {
    ds->count = 0;
}

/* Push a deferred action */
static void xpp_defer_push(XppDeferStack *ds, void (*fn)(void *), void *arg) {
    if (ds->count >= XPP_DEFER_MAX) {
        fprintf(stderr, "runtime error: defer stack overflow (max %d)\n", XPP_DEFER_MAX);
        exit(1);
    }
    ds->entries[ds->count].fn  = fn;
    ds->entries[ds->count].arg = arg;
    ds->count++;
}

/* Run all deferred actions in LIFO order and reset the stack */
static void xpp_defer_run_all(XppDeferStack *ds) {
    for (int i = ds->count - 1; i >= 0; i--) {
        ds->entries[i].fn(ds->entries[i].arg);
    }
    ds->count = 0;
}


/* ============================================================================
 * 11. Math Helpers
 * ============================================================================
 *
 * Thin wrappers around <math.h> so the code generator can emit simple calls.
 */

static double  xpp_math_pow(double base, double exp)  { return pow(base, exp); }
static double  xpp_math_sqrt(double x)                { return sqrt(x); }
static double  xpp_math_floor(double x)               { return floor(x); }
static double  xpp_math_ceil(double x)                { return ceil(x); }
static double  xpp_math_round(double x)               { return round(x); }
static double  xpp_math_sin(double x)                 { return sin(x); }
static double  xpp_math_cos(double x)                 { return cos(x); }
static double  xpp_math_tan(double x)                 { return tan(x); }
static double  xpp_math_log(double x)                 { return log(x); }
static double  xpp_math_log10(double x)               { return log10(x); }
static double  xpp_math_fabs(double x)                { return fabs(x); }
static double  xpp_math_fmod(double x, double y)      { return fmod(x, y); }

static int64_t xpp_math_abs(int64_t x) { return x < 0 ? -x : x; }
static int64_t xpp_math_min(int64_t a, int64_t b) { return a < b ? a : b; }
static int64_t xpp_math_max(int64_t a, int64_t b) { return a > b ? a : b; }

static double  xpp_math_fmin(double a, double b) { return a < b ? a : b; }
static double  xpp_math_fmax(double a, double b) { return a > b ? a : b; }


/* ============================================================================
 * 12. Cleanup  --  Automatic finalization at program exit
 * ============================================================================
 *
 * We use the GCC/Clang __attribute__((destructor)) to run cleanup code
 * automatically when the program exits. On MSVC (which does not support
 * this attribute) we fall back to atexit() via a constructor.
 */

static void xpp_cleanup(void);

#if defined(__GNUC__) || defined(__clang__)

__attribute__((destructor))
static void xpp_cleanup(void) {
    xpp_gc_free_all();
}

#else

/* Fallback for compilers without __attribute__((destructor)) */
static void xpp_cleanup(void) {
    xpp_gc_free_all();
}

/* Register the cleanup function via a constructor or global initializer */
#ifdef _MSC_VER
    /* MSVC: use a CRT initializer section */
    #pragma section(".CRT$XCU", read)
    static void __cdecl xpp_register_cleanup(void) { atexit(xpp_cleanup); }
    __declspec(allocate(".CRT$XCU")) static void (__cdecl *_xpp_init)(void) = xpp_register_cleanup;
#else
    /* Generic fallback: user must call xpp_cleanup() manually or use atexit */
#endif

#endif /* __GNUC__ || __clang__ */


/* ============================================================================
 * End of Xuesos++ Runtime Library
 * ============================================================================ */
