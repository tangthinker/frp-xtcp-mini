// Copyright 2017 fatedier, fatedier@gmail.com
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

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	libio "github.com/fatedier/golib/io"
	libnet "github.com/fatedier/golib/net"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/xlog"
)

var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy, v1.ProxyConfigurer) Proxy{}

func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy, v1.ProxyConfigurer) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

type Proxy interface {
	Run() error
	InWorkConn(net.Conn, *msg.StartWorkConn)
	SetInWorkConnCallback(func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool)
	Close()
}

func NewProxy(
	ctx context.Context,
	pxyConf v1.ProxyConfigurer,
	clientCfg *v1.ClientCommonConfig,
	encryptionKey []byte,
	msgTransporter transport.MessageTransporter,
	udpPacketCodec string,
) (pxy Proxy) {
	baseProxy := BaseProxy{
		baseCfg:        pxyConf.GetBaseConfig(),
		clientCfg:      clientCfg,
		encryptionKey:  encryptionKey,
		msgTransporter: msgTransporter,
		xl:             xlog.FromContextSafe(ctx),
		ctx:            ctx,
		udpPacketCodec: udpPacketCodec,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(pxyConf)]
	if factory == nil {
		return nil
	}
	return factory(&baseProxy, pxyConf)
}

type BaseProxy struct {
	baseCfg            *v1.ProxyBaseConfig
	clientCfg          *v1.ClientCommonConfig
	encryptionKey      []byte
	msgTransporter     transport.MessageTransporter
	inWorkConnCallback func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool

	mu             sync.RWMutex
	xl             *xlog.Logger
	ctx            context.Context
	udpPacketCodec string
}

func (pxy *BaseProxy) Run() error {
	return nil
}

func (pxy *BaseProxy) Close() {}

func (pxy *BaseProxy) wrapWorkConn(conn net.Conn, encKey []byte) (io.ReadWriteCloser, func(), error) {
	var rwc io.ReadWriteCloser = conn
	if pxy.baseCfg.Transport.UseEncryption {
		var err error
		rwc, err = libio.WithEncryption(rwc, encKey)
		if err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("create encryption stream error: %w", err)
		}
	}
	var recycleFn func()
	if pxy.baseCfg.Transport.UseCompression {
		rwc, recycleFn = libio.WithCompressionFromPool(rwc)
	}
	return rwc, recycleFn, nil
}

func (pxy *BaseProxy) SetInWorkConnCallback(cb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool) {
	pxy.inWorkConnCallback = cb
}

func (pxy *BaseProxy) InWorkConn(conn net.Conn, m *msg.StartWorkConn) {
	if pxy.inWorkConnCallback != nil {
		if !pxy.inWorkConnCallback(pxy.baseCfg, conn, m) {
			return
		}
	}
	pxy.HandleTCPWorkConnection(conn, m, pxy.encryptionKey)
}

func (pxy *BaseProxy) HandleTCPWorkConnection(workConn net.Conn, m *msg.StartWorkConn, encKey []byte) {
	xl := pxy.xl
	baseCfg := pxy.baseCfg

	xl.Tracef("handle tcp work connection, useEncryption: %t, useCompression: %t",
		baseCfg.Transport.UseEncryption, baseCfg.Transport.UseCompression)

	remote, recycleFn, err := pxy.wrapWorkConn(workConn, encKey)
	if err != nil {
		xl.Errorf("wrap work connection: %v", err)
		return
	}
	if recycleFn != nil {
		defer recycleFn()
	}

	localConn, err := libnet.Dial(
		net.JoinHostPort(baseCfg.LocalIP, strconv.Itoa(baseCfg.LocalPort)),
		libnet.WithTimeout(10*time.Second),
	)
	if err != nil {
		workConn.Close()
		xl.Errorf("connect to local service [%s:%d] error: %v", baseCfg.LocalIP, baseCfg.LocalPort, err)
		return
	}

	xl.Debugf("join connections, localConn(l[%s] r[%s]) workConn(l[%s] r[%s])", localConn.LocalAddr().String(),
		localConn.RemoteAddr().String(), workConn.LocalAddr().String(), workConn.RemoteAddr().String())

	_, _, errs := libio.Join(localConn, remote)
	xl.Debugf("join connections closed")
	if len(errs) > 0 {
		xl.Tracef("join connections errors: %v", errs)
	}
}
