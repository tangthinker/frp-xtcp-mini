/* Copyright 2026 The frp Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * libfrpc C ABI
 *
 * Build:
 *   go build -buildmode=c-shared  -o libfrpc.so  ./cmd/libfrpc   # linux
 *   go build -buildmode=c-shared  -o libfrpc.dylib ./cmd/libfrpc  # macos
 *   go build -buildmode=c-archive -o libfrpc.a    ./cmd/libfrpc   # ios
 *
 * Strings returned via char** out-params are heap-allocated. Call frpc_free.
 * frpc_version() returns a static string; do not free it.
 *
 * Start is asynchronous: it returns after the config is accepted. Poll
 * frpc_status or call frpc_wait_connected before using visitor URLs.
 *
 * Event callbacks run on a Go background thread. They must return quickly
 * and must not call frpc_stop on the same handle.
 */

#ifndef FRPC_H
#define FRPC_H

#ifdef __cplusplus
extern "C" {
#endif

#define FRPC_API_VERSION 1

#define FRPC_OK 0
#define FRPC_ERR_INVALID_ARG -1
#define FRPC_ERR_INVALID_JSON -2
#define FRPC_ERR_INVALID_CONFIG -3
#define FRPC_ERR_NOT_FOUND -4
#define FRPC_ERR_NOT_CONNECTED -5
#define FRPC_ERR_TIMEOUT -6
#define FRPC_ERR_STOPPED -7
#define FRPC_ERR_FAILED -8
#define FRPC_ERR_INTERNAL -9

/* JSON event/status payload. See libfrpc/README.md for the schema. */
typedef void (*frpc_event_cb)(int handle, const char *json, void *userdata);

/* Start frpc from a JSON/TOML/YAML config string. handle is written on success. */
int frpc_start(const char *config, int *out_handle);

/* Stop the session and invalidate the handle. */
int frpc_stop(int handle);

/* Drop the control connection so the client logs in again. */
int frpc_reconnect(int handle);

/* Block until state is "connected", or until timeout_ms. */
int frpc_wait_connected(int handle, int timeout_ms);

/* Snapshot JSON. Caller must frpc_free(*out_json). */
int frpc_status(int handle, char **out_json);

/* Last error string (may be empty). Caller must frpc_free(*out_error). */
int frpc_last_error(int handle, char **out_error);

/* Subscribe to state-change events. Pass cb=NULL to clear. */
int frpc_set_event_callback(int handle, frpc_event_cb cb, void *userdata);

/* Static version string such as "0.71.0". Do not free. */
const char *frpc_version(void);

/* Human-readable description of a FRPC_ERR_* code. Do not free. */
const char *frpc_strerror(int code);

void frpc_free(char *p);

#ifdef __cplusplus
}
#endif

#endif /* FRPC_H */
