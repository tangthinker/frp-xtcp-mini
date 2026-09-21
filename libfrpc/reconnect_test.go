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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestForcedReconnectIncrementsCount(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)
	first := statusOf(t, handle)
	require.Equal(t, 0, first.ReconnectCount)

	require.NoError(t, Reconnect(handle))
	st := waitReconnectCount(t, handle, 1, 20*time.Second)
	require.Equal(t, StateConnected, st.State)
	require.NotEmpty(t, st.RunID)
	require.Equal(t, first.Visitors[0].BindPort, st.Visitors[0].BindPort)
}

func TestReconnectAfterServerRestart(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)
	require.Equal(t, 0, statusOf(t, handle).ReconnectCount)

	stop()
	waitState(t, handle, StateReconnecting, 10*time.Second)

	_, stop = startFRPSOnPort(t, port, testToken)
	defer stop()

	st := waitReconnectCount(t, handle, 1, 25*time.Second)
	require.Equal(t, StateConnected, st.State)
	require.GreaterOrEqual(t, st.ReconnectCount, 1)
}

func TestMultipleForcedReconnects(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)

	for i := 1; i <= 3; i++ {
		require.NoError(t, Reconnect(handle))
		st := waitReconnectCount(t, handle, i, 20*time.Second)
		require.Equal(t, StateConnected, st.State)
	}
}

func TestConcurrentStatusDuringReconnect(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	defer stop()

	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deadline := time.Now().Add(8 * time.Second)
			for time.Now().Before(deadline) {
				_, _ = GetStatus(handle)
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	require.NoError(t, Reconnect(handle))
	_ = waitReconnectCount(t, handle, 1, 20*time.Second)
	wg.Wait()
}

func TestStopDuringReconnect(t *testing.T) {
	port, stop := startFRPS(t, testToken)
	handle := startHandle(t, visitorJSON(port, ""))
	mustWaitConnected(t, handle)

	stop()
	waitState(t, handle, StateReconnecting, 10*time.Second)
	require.NoError(t, Stop(handle))
	require.Equal(t, CodeNotFound, CodeOf(Stop(handle)))
}
