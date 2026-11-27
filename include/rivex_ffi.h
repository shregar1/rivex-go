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
 *   - Functions returning `char*` (string results): caller must free with
 *     `rivex_free_string()`.
 *   - Functions returning `uint8_t*` with an `out_len` parameter: caller must
 *     free with `rivex_bytes_free(result, out_len)`.
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

typedef struct RivexServerInner  RivexServer;
typedef struct RivexCacheInner   RivexCache;
typedef struct FfiBloomFilterInner  FfiBloomFilter;
typedef struct FfiHyperLogLogInner  FfiHyperLogLog;
typedef struct FfiPriorityQueueInner FfiPriorityQueue;
typedef struct FfiRateLimiterInner  FfiRateLimiter;
typedef struct FfiRingBufferInner   FfiRingBuffer;
typedef struct FfiSortedSetInner    FfiSortedSet;

/* ------------------------------------------------------------------ */
/* Data types exchanged across the FFI boundary                        */
/* ------------------------------------------------------------------ */

typedef struct {
    const char *name;
    const char *value;
} RivexHeaderPair;

typedef struct {
    const char *key;
    const char *value;
} RivexKeyValue;

typedef struct {
    const char           *method;
    const char           *path;
    const char           *query_string;
    const uint8_t        *body;
    size_t                body_len;
    const RivexHeaderPair *headers;
    size_t                headers_len;
    const RivexKeyValue   *path_params;
    size_t                path_params_len;
    const char           *client_host;
    uint16_t              client_port;
} RivexRequestData;

typedef struct {
    uint16_t              status_code;
    const uint8_t        *body;
    size_t                body_len;
    const RivexHeaderPair *headers;
    size_t                headers_len;
} RivexResponseData;

typedef RivexResponseData* (*RivexHandler)(
    const RivexRequestData *request,
    void *user_data
);

/* ------------------------------------------------------------------ */
/* Server lifecycle                                                    */
/* ------------------------------------------------------------------ */

RivexServer* rivex_server_new(void);
void rivex_server_free(RivexServer *server);
int rivex_server_add_route(
    RivexServer *server,
    const char *method,
    const char *path,
    RivexHandler callback,
    void *user_data,
    uint16_t status_code
);
int rivex_server_run(RivexServer *server, const char *host, uint16_t port);

/* ------------------------------------------------------------------ */
/* Memory management                                                   */
/* ------------------------------------------------------------------ */

void rivex_string_free(char *s);
void rivex_response_data_free(RivexResponseData *response);
void rivex_request_data_free(RivexRequestData *request);
void rivex_free_string(char *s);
void rivex_bytes_free(uint8_t *ptr, size_t len);

/* ------------------------------------------------------------------ */
/* Response helpers                                                    */
/* ------------------------------------------------------------------ */

RivexResponseData* rivex_response_json(uint16_t status_code, const char *body);
RivexResponseData* rivex_response_text(uint16_t status_code, const char *body);

/* ------------------------------------------------------------------ */
/* Utilities                                                           */
/* ------------------------------------------------------------------ */

const char* rivex_version(void);

/* ------------------------------------------------------------------ */
/* Cache                                                                */
/* ------------------------------------------------------------------ */

RivexCache* rivex_cache_new(size_t max_size, double ttl_seconds);
void rivex_cache_free(RivexCache *cache);
void rivex_cache_set(RivexCache *cache, const char *key, const char *value, double ttl_seconds);
char* rivex_cache_get(RivexCache *cache, const char *key);
int rivex_cache_delete(RivexCache *cache, const char *key);
void rivex_cache_clear(RivexCache *cache);
size_t rivex_cache_len(RivexCache *cache);
int rivex_cache_contains(RivexCache *cache, const char *key);
char* rivex_cache_keys(RivexCache *cache);
void rivex_cache_evict_expired(RivexCache *cache);

/* ------------------------------------------------------------------ */
/* Bloom Filter                                                         */
/* ------------------------------------------------------------------ */

FfiBloomFilter* rivex_bloom_filter_new(size_t capacity, double fpr);
void rivex_bloom_filter_free(FfiBloomFilter *bf);
void rivex_bloom_filter_add(FfiBloomFilter *bf, const char *item);
int rivex_bloom_filter_contains(FfiBloomFilter *bf, const char *item);
void rivex_bloom_filter_clear(FfiBloomFilter *bf);
size_t rivex_bloom_filter_len(FfiBloomFilter *bf);

/* ------------------------------------------------------------------ */
/* HyperLogLog                                                          */
/* ------------------------------------------------------------------ */

FfiHyperLogLog* rivex_hyperloglog_new(uint8_t precision);
void rivex_hyperloglog_free(FfiHyperLogLog *hll);
void rivex_hyperloglog_add(FfiHyperLogLog *hll, const char *item);
uint64_t rivex_hyperloglog_count(FfiHyperLogLog *hll);
void rivex_hyperloglog_clear(FfiHyperLogLog *hll);

/* ------------------------------------------------------------------ */
/* Priority Queue                                                       */
/* ------------------------------------------------------------------ */

FfiPriorityQueue* rivex_priority_queue_new(void);
void rivex_priority_queue_free(FfiPriorityQueue *pq);
void rivex_priority_queue_push(FfiPriorityQueue *pq, double priority, const char *item);
char* rivex_priority_queue_pop(FfiPriorityQueue *pq);
char* rivex_priority_queue_peek(FfiPriorityQueue *pq);
size_t rivex_priority_queue_len(FfiPriorityQueue *pq);
void rivex_priority_queue_clear(FfiPriorityQueue *pq);

/* ------------------------------------------------------------------ */
/* Sorted Set                                                           */
/* ------------------------------------------------------------------ */

FfiSortedSet* rivex_sorted_set_new(void);
void rivex_sorted_set_free(FfiSortedSet *ss);
void rivex_sorted_set_add(FfiSortedSet *ss, const char *member, double score);
int rivex_sorted_set_remove(FfiSortedSet *ss, const char *member);
double rivex_sorted_set_score(FfiSortedSet *ss, const char *member);
int64_t rivex_sorted_set_rank(FfiSortedSet *ss, const char *member);
char* rivex_sorted_set_top(FfiSortedSet *ss, size_t n);
char* rivex_sorted_set_bottom(FfiSortedSet *ss, size_t n);
size_t rivex_sorted_set_len(FfiSortedSet *ss);
void rivex_sorted_set_clear(FfiSortedSet *ss);

/* ------------------------------------------------------------------ */
/* Ring Buffer                                                          */
/* ------------------------------------------------------------------ */

FfiRingBuffer* rivex_ring_buffer_new(size_t capacity);
void rivex_ring_buffer_free(FfiRingBuffer *rb);
void rivex_ring_buffer_push(FfiRingBuffer *rb, const char *item);
char* rivex_ring_buffer_to_json(FfiRingBuffer *rb);
size_t rivex_ring_buffer_len(FfiRingBuffer *rb);
size_t rivex_ring_buffer_capacity(FfiRingBuffer *rb);
void rivex_ring_buffer_clear(FfiRingBuffer *rb);

/* ------------------------------------------------------------------ */
/* Rate Limiter                                                         */
/* ------------------------------------------------------------------ */

FfiRateLimiter* rivex_rate_limiter_new(double rate, uint32_t burst);
void rivex_rate_limiter_free(FfiRateLimiter *rl);
int rivex_rate_limiter_check(FfiRateLimiter *rl, const char *key);
int rivex_rate_limiter_check_n(FfiRateLimiter *rl, const char *key, uint32_t n);
double rivex_rate_limiter_remaining(FfiRateLimiter *rl, const char *key);
void rivex_rate_limiter_reset(FfiRateLimiter *rl, const char *key);
void rivex_rate_limiter_clear(FfiRateLimiter *rl);

/* ------------------------------------------------------------------ */
/* Compression                                                          */
/* ------------------------------------------------------------------ */

uint8_t* rivex_gzip_compress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_gzip_decompress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_brotli_compress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_brotli_decompress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_zstd_compress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_zstd_decompress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_lz4_compress(const uint8_t *data, size_t len, size_t *out_len);
uint8_t* rivex_lz4_decompress(const uint8_t *data, size_t len, size_t *out_len);

/* ------------------------------------------------------------------ */
/* Hashing / Crypto                                                     */
/* ------------------------------------------------------------------ */

char* rivex_sha256(const uint8_t *data, size_t len);
char* rivex_sha512(const uint8_t *data, size_t len);
char* rivex_hmac_sha256(const uint8_t *key, size_t key_len, const uint8_t *msg, size_t msg_len);
char* rivex_hmac_sha512(const uint8_t *key, size_t key_len, const uint8_t *msg, size_t msg_len);
int rivex_hmac_sha256_verify(const uint8_t *key, size_t key_len, const uint8_t *msg, size_t msg_len, const char *sig_hex);
int rivex_hmac_sha512_verify(const uint8_t *key, size_t key_len, const uint8_t *msg, size_t msg_len, const char *sig_hex);
uint8_t* rivex_aes_gcm_encrypt(const uint8_t *key, size_t key_len, const uint8_t *plaintext, size_t pt_len, size_t *out_len);
uint8_t* rivex_aes_gcm_decrypt(const uint8_t *key, size_t key_len, const uint8_t *data, size_t data_len, size_t *out_len);
uint8_t* rivex_pbkdf2_derive(const uint8_t *password, size_t pw_len, const uint8_t *salt, size_t salt_len, uint32_t iterations, size_t key_length, size_t *out_len);
int rivex_pbkdf2_verify(const uint8_t *password, size_t pw_len, const uint8_t *salt, size_t salt_len, uint32_t iterations, const uint8_t *expected, size_t expected_len);
uint8_t* rivex_random_bytes(size_t n, size_t *out_len);
char* rivex_random_token(size_t n_bytes);
char* rivex_random_hex(size_t n_bytes);
uint8_t* rivex_ed25519_keygen(size_t *out_len);
uint8_t* rivex_ed25519_sign(const uint8_t *private_key, const uint8_t *msg, size_t msg_len, size_t *out_len);
int rivex_ed25519_verify(const uint8_t *public_key, const uint8_t *msg, size_t msg_len, const uint8_t *signature, size_t sig_len);

/* ------------------------------------------------------------------ */
/* Encoding                                                             */
/* ------------------------------------------------------------------ */

char* rivex_b64_encode(const uint8_t *data, size_t len);
uint8_t* rivex_b64_decode(const char *data, size_t *out_len);
char* rivex_jwt_encode(const char *payload_json, const char *secret, int64_t expires_in);
char* rivex_jwt_decode(const char *token, const char *secret, int verify);
char* rivex_parse_cookies(const char *header);
char* rivex_build_set_cookie(const char *name, const char *value, const char *path, int64_t max_age, int secure, int http_only);
char* rivex_parse_query_string(const char *query);
char* rivex_build_query_string(const char *json);

/* ------------------------------------------------------------------ */
/* ID Generation                                                        */
/* ------------------------------------------------------------------ */

char* rivex_uuid_v4(void);
char* rivex_uuid_v7(void);
char* rivex_gen_ulid(void);
uint64_t rivex_ulid_timestamp(const char *ulid);
int64_t rivex_snowflake_id(uint16_t machine_id);
uint64_t rivex_snowflake_timestamp(int64_t id);
uint16_t rivex_snowflake_machine_id(int64_t id);
char* rivex_nano_id(size_t size);
char* rivex_nano_id_custom(size_t size, const char *alphabet);
char* rivex_uuid_v4_bulk(size_t n);
char* rivex_uuid_v7_bulk(size_t n);
char* rivex_ulid_bulk(size_t n);
char* rivex_snowflake_id_bulk(size_t n, uint16_t machine_id);
char* rivex_nano_id_bulk(size_t n, size_t size);

/* ------------------------------------------------------------------ */
/* File & Checksum                                                      */
/* ------------------------------------------------------------------ */

char* rivex_checksum_md5(const uint8_t *data, size_t len);
char* rivex_checksum_sha256(const uint8_t *data, size_t len);
char* rivex_checksum_crc32(const uint8_t *data, size_t len);
uint8_t* rivex_zip_create(const char *entries_json, size_t *out_len);
char* rivex_zip_extract(const uint8_t *data, size_t len);
uint8_t* rivex_image_resize(const uint8_t *data, size_t len, uint32_t width, uint32_t height, size_t *out_len);
uint8_t* rivex_image_thumbnail(const uint8_t *data, size_t len, uint32_t max_dim, size_t *out_len);
int rivex_image_dimensions(const uint8_t *data, size_t len, uint32_t *width, uint32_t *height);
char* rivex_pdf_extract_text(const uint8_t *data, size_t len);

/* ------------------------------------------------------------------ */
/* Password Hashing                                                     */
/* ------------------------------------------------------------------ */

char* rivex_hash_password(const char *password, const char *algorithm);
int rivex_verify_password(const char *password, const char *hash);

/* ------------------------------------------------------------------ */
/* URL / Protocol                                                       */
/* ------------------------------------------------------------------ */

char* rivex_parse_raw_headers(const uint8_t *data, size_t len);
char* rivex_parse_url(const char *url);
char* rivex_build_url(const char *json);
char* rivex_resolve_url(const char *base, const char *relative);
char* rivex_sniff_mime_type(const uint8_t *data, size_t len);
char* rivex_mime_from_extension(const char *ext);
char* rivex_extension_from_mime(const char *mime);
char* rivex_parse_graphql(const char *query);
int rivex_validate_graphql(const char *query);

/* ------------------------------------------------------------------ */
/* Regex                                                                */
/* ------------------------------------------------------------------ */

int rivex_re_match(const char *pattern, const char *text);

/* ------------------------------------------------------------------ */
/* Serialization (YAML, TOML, CSV, MessagePack, XML)                   */
/* ------------------------------------------------------------------ */

char* rivex_yaml_parse(const char *yaml);
char* rivex_yaml_dump(const char *json);
char* rivex_toml_parse(const char *toml);
char* rivex_toml_dump(const char *json);
char* rivex_csv_parse(const char *csv);
char* rivex_csv_dump(const char *json);
uint8_t* rivex_msgpack_encode(const char *json, size_t *out_len);
char* rivex_msgpack_decode(const uint8_t *data, size_t len);
char* rivex_xml_to_json(const char *xml);
char* rivex_json_to_xml(const char *json);

/* ------------------------------------------------------------------ */
/* Text Processing                                                      */
/* ------------------------------------------------------------------ */

char* rivex_slugify(const char *text, const char *separator);
char* rivex_escape_html(const char *text);
char* rivex_markdown_to_html(const char *md);
int rivex_levenshtein_distance(const char *a, const char *b);
char* rivex_strip_html_tags(const char *html);
char* rivex_render_template(const char *tmpl, const char *vars_json);
char* rivex_text_diff(const char *a, const char *b);
double rivex_similarity_score(const char *a, const char *b);
int rivex_fuzzy_match(const char *pattern, const char *text, double threshold);
char* rivex_tokenize_words(const char *text);
char* rivex_tokenize_sentences(const char *text);

/* ------------------------------------------------------------------ */
/* Validation & Sanitization                                            */
/* ------------------------------------------------------------------ */

int rivex_validate_email(const char *email);
int rivex_validate_url(const char *url);
int rivex_luhn_check(const char *number);
int rivex_validate_iban(const char *iban);
char* rivex_sanitize_html(const char *html);

/* ------------------------------------------------------------------ */
/* Middleware Utilities (ETag, CIDR, Webhook, Content-Neg)             */
/* ------------------------------------------------------------------ */

char* rivex_etag_crc32(const uint8_t *data, size_t len);
char* rivex_etag_xxhash(const uint8_t *data, size_t len);
char* rivex_etag_weak(const uint8_t *data, size_t len);
int rivex_etag_match(const char *e1, const char *e2, int32_t strong);
int rivex_ip_in_cidr(const char *ip, const char *cidr);
int rivex_is_private_ip(const char *ip);
char* rivex_parse_cidr(const char *cidr);
char* rivex_body_sha256(const uint8_t *data, size_t len);
char* rivex_webhook_sign(const uint8_t *secret, size_t secret_len, const uint8_t *body, size_t body_len);
int rivex_webhook_verify(const uint8_t *secret, size_t secret_len, const uint8_t *body, size_t body_len, const char *signature);
char* rivex_negotiate_content_type(const char *accept_header, const char *available_json);

/* ------------------------------------------------------------------ */
/* DateTime                                                             */
/* ------------------------------------------------------------------ */

char* rivex_now_iso(void);
char* rivex_cron_next(const char *expr);
int rivex_cron_validate(const char *expr);
char* rivex_convert_timezone(const char *datetime_iso, const char *from_tz, const char *to_tz);

#ifdef __cplusplus
}
#endif

#endif /* RIVEX_FFI_H */
