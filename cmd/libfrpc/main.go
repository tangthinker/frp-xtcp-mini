// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

/*
#include <stdlib.h>
#include <stdint.h>

typedef void (*frpc_event_cb)(int handle, const char *json, void *userdata);

static void frpc_invoke_event_cb(frpc_event_cb cb, int handle, const char *json, void *userdata) {
	if (cb != NULL) {
		cb(handle, json, userdata);
	}
}

// Runs before the Go runtime constructor so Flutter's VM is not preempted by SIGURG.
__attribute__((constructor(101)))
static void frpc_early_godebug(void) {
	setenv("GODEBUG", "asyncpreemptoff=1", 1);
}
*/
import "C"

import (
	"encoding/json"
	"sync"
	"time"
	"unsafe"

	"github.com/fatedier/frp/libfrpc"
)

func main() {}

type cCallback struct {
	cb       C.frpc_event_cb
	userdata unsafe.Pointer
}

var (
	cbMu          sync.Mutex
	callbacks     = map[int32]cCallback{}
	versionOnce   sync.Once
	versionC      *C.char
	strerrorCache sync.Map
)

func storeCallback(handle int32, cb C.frpc_event_cb, userdata unsafe.Pointer) {
	cbMu.Lock()
	defer cbMu.Unlock()
	if cb == nil {
		delete(callbacks, handle)
		return
	}
	callbacks[handle] = cCallback{cb: cb, userdata: userdata}
}

func lookupCallback(handle int32) (cCallback, bool) {
	cbMu.Lock()
	defer cbMu.Unlock()
	cb, ok := callbacks[handle]
	return cb, ok
}

func clearCallback(handle int32) {
	cbMu.Lock()
	delete(callbacks, handle)
	cbMu.Unlock()
}

func emitC(ev libfrpc.Event) {
	cb, ok := lookupCallback(ev.Handle)
	if !ok {
		return
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	cstr := C.CString(string(payload))
	C.frpc_invoke_event_cb(cb.cb, C.int(ev.Handle), cstr, cb.userdata)
	C.free(unsafe.Pointer(cstr))
}

func dupCString(s string) *C.char {
	return C.CString(s)
}

func cVersion() *C.char {
	versionOnce.Do(func() {
		versionC = C.CString(libfrpc.Version())
	})
	return versionC
}

//export frpc_start
func frpc_start(config *C.char, outHandle *C.int) C.int {
	if config == nil || outHandle == nil {
		return C.int(libfrpc.CodeInvalidArg)
	}
	handle, err := libfrpc.Start(C.GoString(config))
	if err != nil {
		return C.int(libfrpc.CodeOf(err))
	}
	*outHandle = C.int(handle)
	return C.int(libfrpc.CodeOK)
}

//export frpc_stop
func frpc_stop(handle C.int) C.int {
	h := int32(handle)
	err := libfrpc.Stop(h)
	clearCallback(h)
	if err != nil {
		return C.int(libfrpc.CodeOf(err))
	}
	return C.int(libfrpc.CodeOK)
}

//export frpc_reconnect
func frpc_reconnect(handle C.int) C.int {
	err := libfrpc.Reconnect(int32(handle))
	return C.int(libfrpc.CodeOf(err))
}

//export frpc_wait_connected
func frpc_wait_connected(handle C.int, timeoutMS C.int) C.int {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	err := libfrpc.WaitConnected(int32(handle), timeout)
	return C.int(libfrpc.CodeOf(err))
}

//export frpc_status
func frpc_status(handle C.int, outJSON **C.char) C.int {
	if outJSON == nil {
		return C.int(libfrpc.CodeInvalidArg)
	}
	js, err := libfrpc.StatusJSON(int32(handle))
	if err != nil {
		return C.int(libfrpc.CodeOf(err))
	}
	*outJSON = dupCString(js)
	return C.int(libfrpc.CodeOK)
}

//export frpc_last_error
func frpc_last_error(handle C.int, outError **C.char) C.int {
	if outError == nil {
		return C.int(libfrpc.CodeInvalidArg)
	}
	msg, err := libfrpc.LastError(int32(handle))
	if err != nil {
		return C.int(libfrpc.CodeOf(err))
	}
	*outError = dupCString(msg)
	return C.int(libfrpc.CodeOK)
}

//export frpc_set_event_callback
func frpc_set_event_callback(handle C.int, cb C.frpc_event_cb, userdata unsafe.Pointer) C.int {
	h := int32(handle)
	storeCallback(h, cb, userdata)
	if cb == nil {
		return C.int(libfrpc.CodeOf(libfrpc.SetEventHandler(h, nil)))
	}
	err := libfrpc.SetEventHandler(h, emitC)
	if err != nil {
		clearCallback(h)
		return C.int(libfrpc.CodeOf(err))
	}
	return C.int(libfrpc.CodeOK)
}

//export frpc_version
func frpc_version() *C.char {
	return cVersion()
}

//export frpc_strerror
func frpc_strerror(code C.int) *C.char {
	key := int32(code)
	if v, ok := strerrorCache.Load(key); ok {
		return v.(*C.char)
	}
	p := C.CString(libfrpc.Code(code).String())
	actual, loaded := strerrorCache.LoadOrStore(key, p)
	if loaded {
		C.free(unsafe.Pointer(p))
		return actual.(*C.char)
	}
	return p
}

//export frpc_free
func frpc_free(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}
