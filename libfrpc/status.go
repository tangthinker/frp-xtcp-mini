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
	"net"
	"strconv"
)

// State is the session lifecycle reported by Status and events.
type State string

const (
	StateStarting     State = "starting"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateReconnecting State = "reconnecting"
	StateStopping     State = "stopping"
	StateStopped      State = "stopped"
	StateFailed       State = "failed"
)

// Endpoint describes a local listener created by a visitor.
type Endpoint struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ServerName string `json:"serverName,omitempty"`
	BindAddr   string `json:"bindAddr"`
	BindPort   int    `json:"bindPort"`
	URL        string `json:"url"`
}

// ProxyInfo is a proxy advertised to frps (home-side role).
type ProxyInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Status is the JSON snapshot returned by StatusJSON / frpc_status.
type Status struct {
	APIVersion     int         `json:"apiVersion"`
	Handle         int32       `json:"handle"`
	State          State       `json:"state"`
	RunID          string      `json:"runId,omitempty"`
	ReconnectCount int         `json:"reconnectCount"`
	LastError      string      `json:"lastError,omitempty"`
	ServerAddr     string      `json:"serverAddr"`
	ServerPort     int         `json:"serverPort"`
	Visitors       []Endpoint  `json:"visitors"`
	Proxies        []ProxyInfo `json:"proxies"`
}

// Event is pushed to EventHandler on every state change. The fields overlap
// with Status so a host can use polling, callbacks, or both.
type Event struct {
	APIVersion     int    `json:"apiVersion"`
	Handle         int32  `json:"handle"`
	State          State  `json:"state"`
	RunID          string `json:"runId,omitempty"`
	ReconnectCount int    `json:"reconnectCount"`
	Error          string `json:"error,omitempty"`
	TS             int64  `json:"ts"`
}

// EventHandler is invoked from a background goroutine. It must not block for
// long, and it must not call Stop on the same handle (Stop waits for shutdown).
type EventHandler func(Event)

func localHTTPURL(bindAddr string, port int) string {
	host := bindAddr
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}
