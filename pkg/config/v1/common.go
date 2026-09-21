// Copyright 2023 The frp Authors
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

package v1

import (
	"github.com/fatedier/frp/pkg/util/util"
)

type AuthScope string

const (
	AuthScopeHeartBeats   AuthScope = "HeartBeats"
	AuthScopeNewWorkConns AuthScope = "NewWorkConns"
)

type AuthMethod string

const (
	AuthMethodToken AuthMethod = "token"
)

type QUICOptions struct {
	KeepalivePeriod    int `json:"keepalivePeriod,omitempty"`
	MaxIdleTimeout     int `json:"maxIdleTimeout,omitempty"`
	MaxIncomingStreams int `json:"maxIncomingStreams,omitempty"`
}

func (c *QUICOptions) Complete() {
	c.KeepalivePeriod = util.EmptyOr(c.KeepalivePeriod, 10)
	c.MaxIdleTimeout = util.EmptyOr(c.MaxIdleTimeout, 30)
	c.MaxIncomingStreams = util.EmptyOr(c.MaxIncomingStreams, 100000)
}

type TLSConfig struct {
	CertFile      string `json:"certFile,omitempty"`
	KeyFile       string `json:"keyFile,omitempty"`
	TrustedCaFile string `json:"trustedCaFile,omitempty"`
	ServerName    string `json:"serverName,omitempty"`
}

type NatTraversalConfig struct {
	DisableAssistedAddrs bool `json:"disableAssistedAddrs,omitempty"`
}

func (c *NatTraversalConfig) Clone() *NatTraversalConfig {
	if c == nil {
		return nil
	}
	out := *c
	return &out
}

type LogConfig struct {
	To                string `json:"to,omitempty"`
	Level             string `json:"level,omitempty"`
	MaxDays           int64  `json:"maxDays"`
	DisablePrintColor bool   `json:"disablePrintColor,omitempty"`
}

func (c *LogConfig) Complete() {
	c.To = util.EmptyOr(c.To, "console")
	c.Level = util.EmptyOr(c.Level, "info")
	c.MaxDays = util.EmptyOr(c.MaxDays, 3)
}
