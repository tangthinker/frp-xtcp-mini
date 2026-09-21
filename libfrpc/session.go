// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or implied, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package libfrpc

import (
	"context"
	"sync"
	"time"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/pkg/config/source"
	"github.com/fatedier/frp/pkg/policy/security"
	"github.com/fatedier/frp/pkg/util/log"
)

const watchInterval = 50 * time.Millisecond

var loggerOnce sync.Once

type session struct {
	handle int32
	svr    *client.Service
	parsed *parsedConfig

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu             sync.Mutex
	state          State
	runID          string
	epoch          uint64
	reconnectCount int
	lastError      string
	stopping       bool
	handler        EventHandler
}

func newSession(parsed *parsedConfig) (*session, error) {
	configSource := source.NewConfigSource()
	if err := configSource.ReplaceAll(parsed.proxies, parsed.visitors); err != nil {
		return nil, wrap(CodeInvalidConfig, err)
	}
	agg := source.NewAggregator(configSource)

	loggerOnce.Do(func() {
		log.InitLogger(parsed.common.Log.To, parsed.common.Log.Level, int(parsed.common.Log.MaxDays), true)
	})

	svr, err := client.NewService(client.ServiceOptions{
		Common:                 parsed.common,
		ConfigSourceAggregator: agg,
		UnsafeFeatures:         security.NewUnsafeFeatures(nil),
	})
	if err != nil {
		return nil, wrap(CodeInternal, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &session{
		svr:    svr,
		parsed: parsed,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		state:  StateStarting,
	}, nil
}

func (s *session) start() {
	go s.run()
}

func (s *session) run() {
	defer close(s.done)
	defer s.cancel()

	s.emitState(StateConnecting, "")
	go s.watchControl()

	err := s.svr.Run(s.ctx)

	s.mu.Lock()
	stopping := s.stopping || s.ctx.Err() != nil
	if stopping {
		s.state = StateStopped
		s.lastError = ""
	} else if err != nil {
		s.state = StateFailed
		s.lastError = err.Error()
	} else {
		s.state = StateStopped
	}
	ev := s.eventLocked()
	h := s.handler
	s.mu.Unlock()
	if h != nil {
		h(ev)
	}
}

func (s *session) watchControl() {
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	wasRunning := false
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			s.onControlTick(s.svr.ControlRunning(), s.svr.ControlEpoch(), &wasRunning)
		}
	}
}

func (s *session) onControlTick(running bool, epoch uint64, wasRunning *bool) {
	var ev *Event
	var h EventHandler

	s.mu.Lock()
	switch s.state {
	case StateStopping, StateStopped, StateFailed:
		s.mu.Unlock()
		return
	}

	if epoch > 0 && epoch != s.epoch {
		if s.epoch != 0 {
			s.reconnectCount++
		}
		s.epoch = epoch
	}

	switch {
	case running:
		s.runID = s.svr.RunID()
		s.lastError = ""
		if s.state != StateConnected {
			s.state = StateConnected
			cp := s.eventLocked()
			ev = &cp
			h = s.handler
		}
		*wasRunning = true
	case *wasRunning:
		s.state = StateReconnecting
		s.lastError = "control connection lost"
		cp := s.eventLocked()
		ev = &cp
		h = s.handler
		*wasRunning = false
	}
	s.mu.Unlock()

	if ev != nil && h != nil {
		h(*ev)
	}
}

func (s *session) emitState(state State, lastError string) {
	s.mu.Lock()
	if s.state == StateStopping || s.state == StateStopped || s.state == StateFailed {
		s.mu.Unlock()
		return
	}
	s.state = state
	s.lastError = lastError
	ev := s.eventLocked()
	h := s.handler
	s.mu.Unlock()
	if h != nil {
		h(ev)
	}
}

func (s *session) eventLocked() Event {
	return Event{
		APIVersion:     APIVersion,
		Handle:         s.handle,
		State:          s.state,
		RunID:          s.runID,
		ReconnectCount: s.reconnectCount,
		Error:          s.lastError,
		TS:             time.Now().UnixMilli(),
	}
}

func (s *session) snapshot() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	visitors := append([]Endpoint(nil), s.parsed.visitorsInfo...)
	proxies := append([]ProxyInfo(nil), s.parsed.proxiesInfo...)
	return Status{
		APIVersion:     APIVersion,
		Handle:         s.handle,
		State:          s.state,
		RunID:          s.runID,
		ReconnectCount: s.reconnectCount,
		LastError:      s.lastError,
		ServerAddr:     s.parsed.common.ServerAddr,
		ServerPort:     s.parsed.common.ServerPort,
		Visitors:       visitors,
		Proxies:        proxies,
	}
}

func (s *session) setHandler(h EventHandler) {
	s.mu.Lock()
	s.handler = h
	s.mu.Unlock()
}

func (s *session) stop() {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		<-s.done
		return
	}
	s.stopping = true
	s.state = StateStopping
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	<-s.done
}

func (s *session) reconnect() error {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	if state != StateConnected && state != StateReconnecting {
		return newError(CodeNotConnected, "session is not connected")
	}
	s.svr.DisconnectControl()
	return nil
}

func (s *session) waitConnected(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		st := s.snapshot()
		switch st.State {
		case StateConnected:
			return nil
		case StateFailed:
			msg := st.LastError
			if msg == "" {
				msg = "session failed"
			}
			return newError(CodeFailed, msg)
		case StateStopped, StateStopping:
			return newError(CodeStopped, "session stopped")
		}
		if time.Now().After(deadline) {
			return newError(CodeTimeout, "wait connected timeout, state="+string(st.State))
		}
		time.Sleep(20 * time.Millisecond)
	}
}
