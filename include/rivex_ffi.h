/*
 * rivex_ffi.h -- C header for the Rivex FFI bindings
 *
 * Include this file from Go (via CGo) or Ruby (via FFI gem) to call into the
 * Rivex Rust HTTP server runtime.
 *
 * Memory management contract:
 *   - Every `rivex_*_new` has a matching `rivex_*_free`.
 *   - `rivex_server_run` consumes the server pointer; do NOT free it after
 *     a successful call.
 *   - Response data returned from `rivex_response_json` / `rivex_response_text`
 *     must be freed with `rivex_response_data_free` unless returned from a
 *     handler callback (in that case Rust frees it).
 *   - `rivex_version` returns a static pointer -- do NOT free it.
 *   - Request data is owned by Rust and freed after the callback returns.
 */

#ifndef RIVEX_FFI_H
#define RIVEX_FFI_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ------------------------------------------------------------------ */
/* Forward declarations / opaque types                                 */
/* ------------------------------------------------------------------ */

/**
 * Opaque server handle.
 *
 * Created by rivex_server_new(), consumed by rivex_server_run(), or freed
 * by rivex_server_free() if the server is never started.
 */
typedef struct RivexServerInner RivexServer;

/* ------------------------------------------------------------------ */
/* Data types exchanged across the FFI boundary                        */
/* ------------------------------------------------------------------ */

/**
 * A single HTTP header as a name/value pair of null-terminated strings.
 */
typedef struct {
    const char *name;
    const char *value;
} RivexHeaderPair;

/**
 * A generic key/value pair (used for path parameters).
 */
typedef struct {
    const char *key;
    const char *value;
} RivexKeyValue;

/**
 * Request data passed to handler callbacks.
 *
 * All pointers are valid only for the duration of the callback invocation.
 * Do NOT stash or free any of these pointers -- Rust manages their lifetime.
 */
typedef struct {
    const char     *method;
    const char     *path;
    const char     *query_string;
    const uint8_t  *body;
    size_t          body_len;
    const RivexHeaderPair *headers;
    size_t          headers_len;
    const RivexKeyValue   *path_params;
    size_t          path_params_len;
    const char     *client_host;
    uint16_t        client_port;
} RivexRequestData;

/**
 * Response data returned from handler callbacks.
 *
 * Allocate this struct (e.g. via malloc or the rivex_response_json/text
 * helpers) and return a pointer from the callback.  Rust copies the data
 * and frees the struct via rivex_response_data_free.
 *
 * If you allocate body/header memory yourself, you are responsible for
 * freeing it AFTER the callback returns (the simplest approach is to use
 * the rivex_response_json / rivex_response_text helpers).
 */
typedef struct {
    uint16_t        status_code;
    const uint8_t  *body;
    size_t          body_len;
    const RivexHeaderPair *headers;
    size_t          headers_len;
} RivexResponseData;

/**
 * Handler callback type.
 *
 * @param request    Pointer to request data (owned by Rust, valid for the
 *                   duration of the call).
 * @param user_data  Opaque pointer registered with rivex_server_add_route.
 * @return           Heap-allocated response data.  Ownership transfers to
 *                   Rust.  Return NULL for a default empty response.
 */
typedef RivexResponseData* (*RivexHandler)(
    const RivexRequestData *request,
    void *user_data
);

/* ------------------------------------------------------------------ */
/* Server lifecycle                                                    */
/* ------------------------------------------------------------------ */

/**
 * Create a new Rivex server builder.
 * @return  Non-null opaque pointer.  Must be freed with rivex_server_free()
 *          or consumed by rivex_server_run().
 */
RivexServer* rivex_server_new(void);

/**
 * Free a server builder that was never started.
 * Passing NULL is a safe no-op.
 */
void rivex_server_free(RivexServer *server);

/**
 * Register a route on the server.
 *
 * @param server       Server builder handle.
 * @param method       HTTP method ("GET", "POST", etc.), null-terminated.
 * @param path         Route pattern, e.g. "/users/{id}", null-terminated.
 * @param callback     Handler function pointer.
 * @param user_data    Opaque context forwarded to the callback.
 * @param status_code  Default status code if the handler returns NULL.
 * @return             0 on success, -1 on error.
 */
int rivex_server_add_route(
    RivexServer *server,
    const char *method,
    const char *path,
    RivexHandler callback,
    void *user_data,
    uint16_t status_code
);

/**
 * Start the server (blocks the calling thread).
 *
 * This consumes the server pointer.  Do NOT call rivex_server_free() after
 * a successful rivex_server_run().
 *
 * @param server  Server builder (consumed on success).
 * @param host    Bind address, e.g. "0.0.0.0", null-terminated.
 * @param port    Port number.
 * @return        0 on clean shutdown, -1 on error.
 */
int rivex_server_run(RivexServer *server, const char *host, uint16_t port);

/* ------------------------------------------------------------------ */
/* Memory management                                                   */
/* ------------------------------------------------------------------ */

/**
 * Free a C string allocated by Rust (e.g. from rivex_version when it is
 * dynamically allocated).  Passing NULL is a safe no-op.
 */
void rivex_string_free(char *s);

/**
 * Free a RivexResponseData allocated by Rust response helpers.
 * Passing NULL is a safe no-op.
 */
void rivex_response_data_free(RivexResponseData *response);

/**
 * Free a RivexRequestData allocated by Rust.
 * Normally not needed -- Rust frees request data after the callback.
 * Passing NULL is a safe no-op.
 */
void rivex_request_data_free(RivexRequestData *request);

/* ------------------------------------------------------------------ */
/* Response helpers                                                    */
/* ------------------------------------------------------------------ */

/**
 * Create a JSON response.
 *
 * Sets Content-Type: application/json.  The returned pointer can be
 * returned directly from a handler callback (Rust will free it), or freed
 * manually with rivex_response_data_free().
 *
 * @param status_code  HTTP status code.
 * @param body         JSON body, null-terminated.
 * @return             Heap-allocated response, or NULL if body is NULL.
 */
RivexResponseData* rivex_response_json(uint16_t status_code, const char *body);

/**
 * Create a plain-text response.
 *
 * Sets Content-Type: text/plain; charset=utf-8.  The returned pointer can
 * be returned directly from a handler callback, or freed manually.
 *
 * @param status_code  HTTP status code.
 * @param body         Text body, null-terminated.
 * @return             Heap-allocated response, or NULL if body is NULL.
 */
RivexResponseData* rivex_response_text(uint16_t status_code, const char *body);

/* ------------------------------------------------------------------ */
/* Utilities                                                           */
/* ------------------------------------------------------------------ */

/**
 * Get the Rivex version string.
 *
 * @return  Static null-terminated string.  Do NOT free this pointer.
 */
const char* rivex_version(void);

#ifdef __cplusplus
}
#endif

#endif /* RIVEX_FFI_H */
