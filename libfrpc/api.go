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

package libfrpc

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/pkg/util/version"
)

var (
	nextHandle atomic.Int32
	sessionsMu sync.RWMutex
	sessions   = map[int32]*session{}
)

func lookup(handle int32) (*session, error) {
	sessionsMu.RLock()
	s := sessions[handle]
	sessionsMu.RUnlock()
	if s == nil {
		return nil, newError(CodeNotFound, "handle not found")
	}
	return s, nil
}

func register(s *session) int32 {
	handle := nextHandle.Add(1)
	s.handle = handle
	sessionsMu.Lock()
	sessions[handle] = s
	sessionsMu.Unlock()
	return handle
}

func unregister(handle int32) {
	sessionsMu.Lock()
	delete(sessions, handle)
	sessionsMu.Unlock()
}

// Start parses config (JSON / TOML / YAML), starts frpc in the background, and
// returns a handle. It does not wait for login; call WaitConnected or poll GetStatus.
func Start(config string) (int32, error) {
	parsed, err := parseClientConfig(config)
	if err != nil {
		return 0, err
	}
	s, err := newSession(parsed)
	if err != nil {
		return 0, err
	}
	handle := register(s)
	s.start()
	return handle, nil
}

// Stop shuts down the session and releases the handle. It is idempotent for a
// live handle; a missing handle is an error.
func Stop(handle int32) error {
	s, err := lookup(handle)
	if err != nil {
		return err
	}
	s.stop()
	unregister(handle)
	return nil
}

// Reconnect drops the current control connection so frpc logs in again.
func Reconnect(handle int32) error {
	s, err := lookup(handle)
	if err != nil {
		return err
	}
	return s.reconnect()
}

// GetStatus returns the current snapshot.
func GetStatus(handle int32) (Status, error) {
	s, err := lookup(handle)
	if err != nil {
		return Status{}, err
	}
	return s.snapshot(), nil
}

// StatusJSON returns the snapshot encoded as JSON.
func StatusJSON(handle int32) (string, error) {
	st, err := GetStatus(handle)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(st)
	if err != nil {
		return "", wrap(CodeInternal, err)
	}
	return string(b), nil
}

// WaitConnected blocks until state is connected, the session fails/stops, or timeout.
func WaitConnected(handle int32, timeout time.Duration) error {
	s, err := lookup(handle)
	if err != nil {
		return err
	}
	return s.waitConnected(timeout)
}

// SetEventHandler installs a callback for state changes. Pass nil to clear.
func SetEventHandler(handle int32, h EventHandler) error {
	s, err := lookup(handle)
	if err != nil {
		return err
	}
	s.setHandler(h)
	return nil
}

// LastError returns the last error string for a handle (may be empty).
func LastError(handle int32) (string, error) {
	st, err := GetStatus(handle)
	if err != nil {
		return "", err
	}
	return st.LastError, nil
}

// Version returns the frp version string.
func Version() string {
	return version.Full()
}
