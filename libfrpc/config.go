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
	"fmt"
	"net"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
)

type parsedConfig struct {
	common       *v1.ClientCommonConfig
	proxies      []v1.ProxyConfigurer
	visitors     []v1.VisitorConfigurer
	visitorsInfo []Endpoint
	proxiesInfo  []ProxyInfo
}

func parseClientConfig(raw string) (*parsedConfig, error) {
	if raw == "" {
		return nil, newError(CodeInvalidArg, "config is empty")
	}

	allCfg := v1.ClientConfig{}
	if err := config.LoadConfigure([]byte(raw), &allCfg, true); err != nil {
		return nil, wrap(CodeInvalidJSON, err)
	}

	common := &allCfg.ClientCommonConfig
	proxies := make([]v1.ProxyConfigurer, 0, len(allCfg.Proxies))
	for _, c := range allCfg.Proxies {
		proxies = append(proxies, c.ProxyConfigurer)
	}
	visitors := make([]v1.VisitorConfigurer, 0, len(allCfg.Visitors))
	for _, c := range allCfg.Visitors {
		visitors = append(visitors, c.VisitorConfigurer)
	}

	if len(common.IncludeConfigFiles) > 0 {
		extProxies, extVisitors, err := config.LoadAdditionalClientConfigs(common.IncludeConfigFiles, false, true)
		if err != nil {
			return nil, wrap(CodeInvalidConfig, err)
		}
		proxies = append(proxies, extProxies...)
		visitors = append(visitors, extVisitors...)
	}

	applyEmbedDefaults(common)
	if err := common.Complete(); err != nil {
		return nil, wrap(CodeInvalidConfig, err)
	}

	proxies, visitors = config.FilterClientConfigurers(common, proxies, visitors)
	proxies = config.CompleteProxyConfigurers(proxies)
	visitors = config.CompleteVisitorConfigurers(visitors)

	if err := allocateVisitorPorts(visitors); err != nil {
		return nil, wrap(CodeInvalidConfig, err)
	}

	if _, err := validation.ValidateAllClientConfig(common, proxies, visitors, security.NewUnsafeFeatures(nil)); err != nil {
		return nil, wrap(CodeInvalidConfig, err)
	}

	parsed := &parsedConfig{
		common:       common,
		proxies:      proxies,
		visitors:     visitors,
		visitorsInfo: make([]Endpoint, 0, len(visitors)),
		proxiesInfo:  make([]ProxyInfo, 0, len(proxies)),
	}
	for _, v := range visitors {
		base := v.GetBaseConfig()
		parsed.visitorsInfo = append(parsed.visitorsInfo, Endpoint{
			Name:       base.Name,
			Type:       base.Type,
			ServerName: base.ServerName,
			BindAddr:   base.BindAddr,
			BindPort:   base.BindPort,
			URL:        localHTTPURL(base.BindAddr, base.BindPort),
		})
	}
	for _, p := range proxies {
		base := p.GetBaseConfig()
		parsed.proxiesInfo = append(parsed.proxiesInfo, ProxyInfo{Name: base.Name, Type: base.Type})
	}
	return parsed, nil
}

func applyEmbedDefaults(c *v1.ClientCommonConfig) {
	if c == nil {
		return
	}
	// Keep retrying after control loss; a mobile host should not exit on the first failure.
	if c.LoginFailExit == nil {
		c.LoginFailExit = lo.ToPtr(false)
	}
}

func allocateVisitorPorts(visitors []v1.VisitorConfigurer) error {
	for _, v := range visitors {
		base := v.GetBaseConfig()
		if base.BindPort != 0 {
			continue
		}
		port, err := allocateTCPPort(base.BindAddr)
		if err != nil {
			return fmt.Errorf("visitor %s: allocate bind port: %w", base.Name, err)
		}
		base.BindPort = port
	}
	return nil
}

func allocateTCPPort(bindAddr string) (int, error) {
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(bindAddr, "0"))
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener addr %T", ln.Addr())
	}
	return addr.Port, nil
}
