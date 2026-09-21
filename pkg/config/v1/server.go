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
	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/util/util"
)

type ServerConfig struct {
	APIMetadata

	Auth AuthServerConfig `json:"auth,omitempty"`
	BindAddr string `json:"bindAddr,omitempty"`
	BindPort int    `json:"bindPort,omitempty"`

	Log LogConfig `json:"log,omitempty"`

	Transport ServerTransportConfig `json:"transport,omitempty"`

	DetailedErrorsToClient *bool `json:"detailedErrorsToClient,omitempty"`
	UserConnTimeout        int64 `json:"userConnTimeout,omitempty"`
	NatHoleAnalysisDataReserveHours int64 `json:"natholeAnalysisDataReserveHours,omitempty"`
}

func (c *ServerConfig) Complete() error {
	if err := c.Auth.Complete(); err != nil {
		return err
	}
	c.Log.Complete()
	c.Transport.Complete()

	c.BindAddr = util.EmptyOr(c.BindAddr, "0.0.0.0")
	c.BindPort = util.EmptyOr(c.BindPort, 7000)
	c.DetailedErrorsToClient = util.EmptyOr(c.DetailedErrorsToClient, lo.ToPtr(true))
	c.UserConnTimeout = util.EmptyOr(c.UserConnTimeout, 10)
	c.NatHoleAnalysisDataReserveHours = util.EmptyOr(c.NatHoleAnalysisDataReserveHours, 7*24)
	return nil
}

type AuthServerConfig struct {
	Method           AuthMethod  `json:"method,omitempty"`
	AdditionalScopes []AuthScope `json:"additionalScopes,omitempty"`
	Token            string      `json:"token,omitempty"`
	TokenSource      *ValueSource `json:"tokenSource,omitempty"`
}

func (c *AuthServerConfig) Complete() error {
	c.Method = util.EmptyOr(c.Method, "token")
	return nil
}

type ServerTransportConfig struct {
	TCPMux                  *bool `json:"tcpMux,omitempty"`
	TCPMuxKeepaliveInterval int64 `json:"tcpMuxKeepaliveInterval,omitempty"`
	TCPKeepAlive            int64 `json:"tcpKeepalive,omitempty"`
	MaxPoolCount            int64 `json:"maxPoolCount,omitempty"`
	HeartbeatTimeout        int64 `json:"heartbeatTimeout,omitempty"`
	QUIC                    QUICOptions `json:"quic,omitempty"`
	TLS                     TLSServerConfig `json:"tls,omitempty"`
}

func (c *ServerTransportConfig) Complete() {
	c.TCPMux = util.EmptyOr(c.TCPMux, lo.ToPtr(true))
	c.TCPMuxKeepaliveInterval = util.EmptyOr(c.TCPMuxKeepaliveInterval, 30)
	c.TCPKeepAlive = util.EmptyOr(c.TCPKeepAlive, 7200)
	c.MaxPoolCount = util.EmptyOr(c.MaxPoolCount, 5)
	if lo.FromPtr(c.TCPMux) {
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, -1)
	} else {
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, 90)
	}
	c.QUIC.Complete()
	if c.TLS.TrustedCaFile != "" {
		c.TLS.Force = true
	}
}

type TLSServerConfig struct {
	Force bool `json:"force,omitempty"`
	TLSConfig
}
