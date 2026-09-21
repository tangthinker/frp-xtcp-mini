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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartWaitConnectedAndStatus(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)

	st := statusOf(t, handle)
	require.Equal(t, StateConnected, st.State)
	require.Equal(t, APIVersion, st.APIVersion)
	require.Equal(t, handle, st.Handle)
	require.NotEmpty(t, st.RunID)
	require.Equal(t, 0, st.ReconnectCount)
	require.Equal(t, "127.0.0.1", st.ServerAddr)
	require.Equal(t, port, st.ServerPort)
	require.Len(t, st.Visitors, 1)
	require.Greater(t, st.Visitors[0].BindPort, 0)
	require.Contains(t, st.Visitors[0].URL, "http://127.0.0.1:")

	raw, err := StatusJSON(handle)
	require.NoError(t, err)
	var decoded Status
	require.NoError(t, json.Unmarshal([]byte(raw), &decoded))
	require.Equal(t, st.RunID, decoded.RunID)
}

func TestProxyAndVisitorLoginTogether(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	localPort := freeTCPPort(t)
	proxyHandle := startHandle(t, proxyJSON(port, localPort))
	visitorHandle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, proxyHandle)
	mustWaitConnected(t, visitorHandle)

	require.Equal(t, "ssh", statusOf(t, proxyHandle).Proxies[0].Name)
	require.Equal(t, "ssh-visitor", statusOf(t, visitorHandle).Visitors[0].Name)
}

func TestStartStopStart(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)
	require.NoError(t, Stop(handle))
	require.Equal(t, CodeNotFound, CodeOf(Stop(handle)))

	handle2 := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle2)
	require.NotEqual(t, handle, handle2)
}

func TestWaitConnectedTimeoutWhileConnecting(t *testing.T) {
	handle := startHandle(t, visitorJSON(1, `"loginFailExit": false,`))
	err := WaitConnected(handle, 400*time.Millisecond)
	require.Equal(t, CodeTimeout, CodeOf(err))
	require.Equal(t, StateConnecting, statusOf(t, handle).State)
}

func TestWaitConnectedFailedWhenLoginFailExit(t *testing.T) {
	handle := startHandle(t, visitorJSON(1, `"loginFailExit": true,`))
	err := WaitConnected(handle, 8*time.Second)
	require.Equal(t, CodeFailed, CodeOf(err))
	require.Equal(t, StateFailed, statusOf(t, handle).State)
	require.NotEmpty(t, statusOf(t, handle).LastError)
}

func TestReconnectRejectedBeforeConnected(t *testing.T) {
	handle := startHandle(t, visitorJSON(1, `"loginFailExit": false,`))
	err := Reconnect(handle)
	require.Equal(t, CodeNotConnected, CodeOf(err))
}

func TestUnknownHandle(t *testing.T) {
	require.Equal(t, CodeNotFound, CodeOf(Stop(99999)))
	_, err := GetStatus(99999)
	require.Equal(t, CodeNotFound, CodeOf(err))
	require.Equal(t, CodeNotFound, CodeOf(WaitConnected(99999, time.Second)))
}

func TestStartInvalidConfig(t *testing.T) {
	_, err := Start("{")
	require.Equal(t, CodeInvalidJSON, CodeOf(err))
	_, err = Start("")
	require.Equal(t, CodeInvalidArg, CodeOf(err))
}

func TestEventHandlerSeesConnected(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle, err := Start(visitorJSON(port, ""))
	require.NoError(t, err)
	t.Cleanup(func() { _ = Stop(handle) })

	var got atomic.Value
	require.NoError(t, SetEventHandler(handle, func(ev Event) {
		if ev.State == StateConnected {
			got.Store(ev)
		}
	}))
	mustWaitConnected(t, handle)

	deadline := time.Now().Add(2 * time.Second)
	for got.Load() == nil {
		if time.Now().After(deadline) {
			t.Fatal("did not receive connected event")
		}
		time.Sleep(20 * time.Millisecond)
	}
	ev := got.Load().(Event)
	require.Equal(t, handle, ev.Handle)
	require.Equal(t, APIVersion, ev.APIVersion)
	require.NotEmpty(t, ev.RunID)
}

func TestConcurrentStatusDuringLogin(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle := startHandle(t, visitorJSON(port, ""))
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 30 {
				_, _ = StatusJSON(handle)
			}
		}()
	}
	mustWaitConnected(t, handle)
	wg.Wait()
}

func TestVersionNonEmpty(t *testing.T) {
	require.NotEmpty(t, Version())
}
