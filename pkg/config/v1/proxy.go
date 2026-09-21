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
	"maps"
	"reflect"
	"slices"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/jsonx"
	"github.com/fatedier/frp/pkg/util/util"
)

type ProxyTransport struct {
	UseEncryption  bool `json:"useEncryption,omitempty"`
	UseCompression bool `json:"useCompression,omitempty"`
}

type ProxyBackend struct {
	LocalIP   string `json:"localIP,omitempty"`
	LocalPort int    `json:"localPort,omitempty"`
}

type ProxyBaseConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Transport   ProxyTransport    `json:"transport,omitempty"`
	Metadatas   map[string]string `json:"metadatas,omitempty"`
	ProxyBackend
}

func (c ProxyBaseConfig) Clone() ProxyBaseConfig {
	out := c
	out.Enabled = util.ClonePtr(c.Enabled)
	out.Annotations = maps.Clone(c.Annotations)
	out.Metadatas = maps.Clone(c.Metadatas)
	return out
}

func (c *ProxyBaseConfig) GetBaseConfig() *ProxyBaseConfig {
	return c
}

func (c *ProxyBaseConfig) Complete() {
	c.LocalIP = util.EmptyOr(c.LocalIP, "127.0.0.1")
}

func (c *ProxyBaseConfig) MarshalToMsg(m *msg.NewProxy) {
	m.ProxyName = c.Name
	m.ProxyType = c.Type
	m.UseEncryption = c.Transport.UseEncryption
	m.UseCompression = c.Transport.UseCompression
	m.Metas = c.Metadatas
	m.Annotations = c.Annotations
}

func (c *ProxyBaseConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.Name = m.ProxyName
	c.Type = m.ProxyType
	c.Transport.UseEncryption = m.UseEncryption
	c.Transport.UseCompression = m.UseCompression
	c.Metadatas = m.Metas
	c.Annotations = m.Annotations
}

type TypedProxyConfig struct {
	Type string `json:"type"`
	ProxyConfigurer
}

func (c *TypedProxyConfig) UnmarshalJSON(b []byte) error {
	configurer, err := DecodeProxyConfigurerJSON(b, DecodeOptions{})
	if err != nil {
		return err
	}

	c.Type = configurer.GetBaseConfig().Type
	c.ProxyConfigurer = configurer
	return nil
}

func (c *TypedProxyConfig) MarshalJSON() ([]byte, error) {
	return jsonx.Marshal(c.ProxyConfigurer)
}

type ProxyConfigurer interface {
	Complete()
	GetBaseConfig() *ProxyBaseConfig
	Clone() ProxyConfigurer
	MarshalToMsg(*msg.NewProxy)
	UnmarshalFromMsg(*msg.NewProxy)
}

type ProxyType string

const (
	ProxyTypeXTCP ProxyType = "xtcp"
)

var proxyConfigTypeMap = map[ProxyType]reflect.Type{
	ProxyTypeXTCP: reflect.TypeFor[XTCPProxyConfig](),
}

func NewProxyConfigurerByType(proxyType ProxyType) ProxyConfigurer {
	v, ok := proxyConfigTypeMap[proxyType]
	if !ok {
		return nil
	}
	pc := reflect.New(v).Interface().(ProxyConfigurer)
	pc.GetBaseConfig().Type = string(proxyType)
	return pc
}

var _ ProxyConfigurer = &XTCPProxyConfig{}

type XTCPProxyConfig struct {
	ProxyBaseConfig

	Secretkey  string   `json:"secretKey,omitempty"`
	AllowUsers []string `json:"allowUsers,omitempty"`

	NatTraversal *NatTraversalConfig `json:"natTraversal,omitempty"`
}

func (c *XTCPProxyConfig) MarshalToMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.MarshalToMsg(m)
	m.Sk = c.Secretkey
	m.AllowUsers = c.AllowUsers
}

func (c *XTCPProxyConfig) UnmarshalFromMsg(m *msg.NewProxy) {
	c.ProxyBaseConfig.UnmarshalFromMsg(m)
	c.Secretkey = m.Sk
	c.AllowUsers = m.AllowUsers
}

func (c *XTCPProxyConfig) Clone() ProxyConfigurer {
	out := *c
	out.ProxyBaseConfig = c.ProxyBaseConfig.Clone()
	out.AllowUsers = slices.Clone(c.AllowUsers)
	out.NatTraversal = c.NatTraversal.Clone()
	return &out
}
