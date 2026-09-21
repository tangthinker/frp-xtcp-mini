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
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/server"
)

const testToken = "libfrpc-test-token"

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

func waitTCP(t *testing.T, host string, port int) {
	t.Helper()
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s: %v", addr, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func startFRPS(t *testing.T, token string) (int, func()) {
	t.Helper()
	return startFRPSOnPort(t, freeTCPPort(t), token)
}

func startFRPSOnPort(t *testing.T, port int, token string) (int, func()) {
	t.Helper()
	cfg := &v1.ServerConfig{
		BindAddr: "127.0.0.1",
		BindPort: port,
	}
	cfg.Auth.Method = "token"
	cfg.Auth.Token = token
	require.NoError(t, cfg.Complete())

	var svr *server.Service
	var err error
	deadline := time.Now().Add(3 * time.Second)
	for {
		svr, err = server.NewService(cfg)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			require.NoError(t, err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	done := make(chan struct{})
	go func() {
		svr.Run(context.Background())
		close(done)
	}()
	waitTCP(t, "127.0.0.1", port)

	stop := func() {
		_ = svr.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("frps did not stop")
		}
	}
	return port, stop
}

func visitorJSON(serverPort int, extra string) string {
	if extra == "" {
		extra = `"loginFailExit": false,`
	}
	return fmt.Sprintf(`{
		"serverAddr": "127.0.0.1",
		"serverPort": %d,
		%s
		"auth": {"method": "token", "token": %q},
		"log": {"to": "console", "level": "error", "disablePrintColor": true},
		"transport": {"dialServerTimeout": 2},
		"visitors": [{
			"name": "ssh-visitor",
			"type": "xtcp",
			"serverName": "ssh",
			"secretKey": "abcdefg",
			"bindAddr": "127.0.0.1",
			"bindPort": 0,
			"keepTunnelOpen": true
		}]
	}`, serverPort, extra, testToken)
}

func proxyJSON(serverPort, localPort int) string {
	return fmt.Sprintf(`{
		"serverAddr": "127.0.0.1",
		"serverPort": %d,
		"auth": {"method": "token", "token": %q},
		"log": {"to": "console", "level": "error", "disablePrintColor": true},
		"transport": {"dialServerTimeout": 2},
		"proxies": [{
			"name": "ssh",
			"type": "xtcp",
			"secretKey": "abcdefg",
			"localIP": "127.0.0.1",
			"localPort": %d
		}]
	}`, serverPort, testToken, localPort)
}

func startHandle(t *testing.T, cfg string) int32 {
	t.Helper()
	handle, err := Start(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = Stop(handle) })
	return handle
}

func mustWaitConnected(t *testing.T, handle int32) {
	t.Helper()
	require.NoError(t, WaitConnected(handle, 15*time.Second))
}

func statusOf(t *testing.T, handle int32) Status {
	t.Helper()
	st, err := GetStatus(handle)
	require.NoError(t, err)
	return st
}

func waitReconnectCount(t *testing.T, handle int32, min int, timeout time.Duration) Status {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var st Status
	var err error
	for {
		st, err = GetStatus(handle)
		require.NoError(t, err)
		if st.ReconnectCount >= min && st.State == StateConnected {
			return st
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting reconnectCount>=%d connected, last=%+v", min, st)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func waitState(t *testing.T, handle int32, want State, timeout time.Duration) Status {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		st := statusOf(t, handle)
		if st.State == want {
			return st
		}
		if time.Now().After(deadline) {
			raw, _ := json.Marshal(st)
			t.Fatalf("timed out waiting state %s, last=%s", want, raw)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
