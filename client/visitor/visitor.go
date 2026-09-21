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

package visitor

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	libio "github.com/fatedier/golib/io"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type Helper interface {
	ConnectServer() (*msg.Conn, error)
	TransferConn(string, net.Conn) error
	MsgTransporter() transport.MessageTransporter
	RunID() string
}

type Visitor interface {
	Run() error
	AcceptConn(conn net.Conn) error
	Close()
}

func NewVisitor(
	ctx context.Context,
	cfg v1.VisitorConfigurer,
	clientCfg *v1.ClientCommonConfig,
	helper Helper,
) (Visitor, error) {
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(cfg.GetBaseConfig().Name)
	ctx = xlog.NewContext(ctx, xl)
	baseVisitor := BaseVisitor{
		clientCfg:  clientCfg,
		helper:     helper,
		ctx:        ctx,
		internalLn: netpkg.NewInternalListener(),
	}
	switch cfg := cfg.(type) {
	case *v1.XTCPVisitorConfig:
		return &XTCPVisitor{
			BaseVisitor:   &baseVisitor,
			cfg:           cfg,
			startTunnelCh: make(chan struct{}),
		}, nil
	default:
		return nil, nil
	}
}

type BaseVisitor struct {
	clientCfg  *v1.ClientCommonConfig
	helper     Helper
	l          net.Listener
	internalLn *netpkg.InternalListener

	mu  sync.RWMutex
	ctx context.Context
}

func (v *BaseVisitor) AcceptConn(conn net.Conn) error {
	return v.internalLn.PutConn(conn)
}

func (v *BaseVisitor) acceptLoop(l net.Listener, name string, handleConn func(net.Conn)) {
	xl := xlog.FromContextSafe(v.ctx)
	for {
		conn, err := l.Accept()
		if err != nil {
			xl.Warnf("%s listener closed", name)
			return
		}
		go handleConn(conn)
	}
}

func (v *BaseVisitor) Close() {
	if v.l != nil {
		v.l.Close()
	}
	if v.internalLn != nil {
		v.internalLn.Close()
	}
}

func wrapVisitorConn(conn io.ReadWriteCloser, cfg *v1.VisitorBaseConfig) (io.ReadWriteCloser, func(), error) {
	rwc := conn
	if cfg.Transport.UseEncryption {
		var err error
		rwc, err = libio.WithEncryption(rwc, []byte(cfg.SecretKey))
		if err != nil {
			return nil, func() {}, fmt.Errorf("create encryption stream error: %v", err)
		}
	}
	recycleFn := func() {}
	if cfg.Transport.UseCompression {
		rwc, recycleFn = libio.WithCompressionFromPool(rwc)
	}
	return rwc, recycleFn, nil
}
