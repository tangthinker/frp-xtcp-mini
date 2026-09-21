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
	"os"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/util/util"
)

type ClientConfig struct {
	ClientCommonConfig

	Proxies  []TypedProxyConfig   `json:"proxies,omitempty"`
	Visitors []TypedVisitorConfig `json:"visitors,omitempty"`
}

type ClientCommonConfig struct {
	APIMetadata

	Auth     AuthClientConfig `json:"auth,omitempty"`
	User     string           `json:"user,omitempty"`
	ClientID string           `json:"clientID,omitempty"`

	ServerAddr        string `json:"serverAddr,omitempty"`
	ServerPort        int    `json:"serverPort,omitempty"`
	NatHoleSTUNServer string `json:"natHoleStunServer,omitempty"`
	DNSServer         string `json:"dnsServer,omitempty"`
	LoginFailExit     *bool  `json:"loginFailExit,omitempty"`
	Start             []string `json:"start,omitempty"`

	Log       LogConfig             `json:"log,omitempty"`
	Transport ClientTransportConfig `json:"transport,omitempty"`

	Metadatas          map[string]string `json:"metadatas,omitempty"`
	IncludeConfigFiles []string          `json:"includes,omitempty"`
	Store              StoreConfig       `json:"store,omitempty"`
}

func (c *ClientCommonConfig) Complete() error {
	c.ServerAddr = util.EmptyOr(c.ServerAddr, "0.0.0.0")
	c.ServerPort = util.EmptyOr(c.ServerPort, 7000)
	c.LoginFailExit = util.EmptyOr(c.LoginFailExit, lo.ToPtr(true))
	c.NatHoleSTUNServer = util.EmptyOr(c.NatHoleSTUNServer, "stun.easyvoip.com:3478")

	if err := c.Auth.Complete(); err != nil {
		return err
	}
	c.Log.Complete()
	c.Transport.Complete()
	return nil
}

type ClientTransportConfig struct {
	Protocol                string `json:"protocol,omitempty"`
	WireProtocol            string `json:"wireProtocol,omitempty"`
	DialServerTimeout       int64  `json:"dialServerTimeout,omitempty"`
	DialServerKeepAlive     int64  `json:"dialServerKeepalive,omitempty"`
	ConnectServerLocalIP    string `json:"connectServerLocalIP,omitempty"`
	ProxyURL                string `json:"proxyURL,omitempty"`
	PoolCount               int    `json:"poolCount,omitempty"`
	TCPMux                  *bool  `json:"tcpMux,omitempty"`
	TCPMuxKeepaliveInterval int64  `json:"tcpMuxKeepaliveInterval,omitempty"`
	QUIC                    QUICOptions `json:"quic,omitempty"`
	HeartbeatInterval       int64  `json:"heartbeatInterval,omitempty"`
	HeartbeatTimeout        int64  `json:"heartbeatTimeout,omitempty"`
	TLS                     TLSClientConfig `json:"tls,omitempty"`
}

func (c *ClientTransportConfig) Complete() {
	c.Protocol = util.EmptyOr(c.Protocol, "tcp")
	c.WireProtocol = util.EmptyOr(c.WireProtocol, "v1")
	c.DialServerTimeout = util.EmptyOr(c.DialServerTimeout, 10)
	c.DialServerKeepAlive = util.EmptyOr(c.DialServerKeepAlive, 7200)
	c.ProxyURL = util.EmptyOr(c.ProxyURL, os.Getenv("http_proxy"))
	c.PoolCount = util.EmptyOr(c.PoolCount, 1)
	c.TCPMux = util.EmptyOr(c.TCPMux, lo.ToPtr(true))
	c.TCPMuxKeepaliveInterval = util.EmptyOr(c.TCPMuxKeepaliveInterval, 30)
	if lo.FromPtr(c.TCPMux) {
		c.HeartbeatInterval = util.EmptyOr(c.HeartbeatInterval, -1)
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, -1)
	} else {
		c.HeartbeatInterval = util.EmptyOr(c.HeartbeatInterval, 30)
		c.HeartbeatTimeout = util.EmptyOr(c.HeartbeatTimeout, 90)
	}
	c.QUIC.Complete()
	c.TLS.Complete()
}

type TLSClientConfig struct {
	Enable                    *bool `json:"enable,omitempty"`
	DisableCustomTLSFirstByte *bool `json:"disableCustomTLSFirstByte,omitempty"`
	TLSConfig
}

func (c *TLSClientConfig) Complete() {
	c.Enable = util.EmptyOr(c.Enable, lo.ToPtr(true))
	c.DisableCustomTLSFirstByte = util.EmptyOr(c.DisableCustomTLSFirstByte, lo.ToPtr(true))
}

type AuthClientConfig struct {
	Method           AuthMethod   `json:"method,omitempty"`
	AdditionalScopes []AuthScope  `json:"additionalScopes,omitempty"`
	Token            string       `json:"token,omitempty"`
	TokenSource      *ValueSource `json:"tokenSource,omitempty"`
}

func (c *AuthClientConfig) Complete() error {
	c.Method = util.EmptyOr(c.Method, "token")
	return nil
}
