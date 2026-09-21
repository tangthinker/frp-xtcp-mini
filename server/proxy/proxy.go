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
	"net"
	"reflect"
	"strconv"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/auth"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
	"github.com/fatedier/frp/server/controller"
)

var proxyFactoryRegistry = map[reflect.Type]func(*BaseProxy) Proxy{}

func RegisterProxyFactory(proxyConfType reflect.Type, factory func(*BaseProxy) Proxy) {
	proxyFactoryRegistry[proxyConfType] = factory
}

type WorkConn struct {
	conn *msg.Conn
}

func NewWorkConn(conn *msg.Conn) *WorkConn {
	return &WorkConn{conn: conn}
}

func (c *WorkConn) Start(m *msg.StartWorkConn) (net.Conn, error) {
	if err := c.conn.WriteMsg(m); err != nil {
		return nil, err
	}
	return c.conn, nil
}

func (c *WorkConn) Close() error {
	return c.conn.Close()
}

type GetWorkConnFn func() (*WorkConn, error)

type Proxy interface {
	Context() context.Context
	Run() (remoteAddr string, err error)
	GetName() string
	GetConfigurer() v1.ProxyConfigurer
	GetWorkConnFromPool(src, dst net.Addr) (workConn net.Conn, err error)
	GetUsedPortsNum() int
	GetResourceController() *controller.ResourceController
	GetUserInfo() auth.UserInfo
	GetLoginMsg() *msg.Login
	Close()
}

type BaseProxy struct {
	name          string
	rc            *controller.ResourceController
	usedPortsNum  int
	poolCount     int
	getWorkConnFn GetWorkConnFn
	serverCfg     *v1.ServerConfig
	encryptionKey []byte
	userInfo      auth.UserInfo
	loginMsg      *msg.Login
	configurer    v1.ProxyConfigurer
	wireProtocol  string

	mu  sync.RWMutex
	xl  *xlog.Logger
	ctx context.Context
}

func (pxy *BaseProxy) GetName() string {
	return pxy.name
}

func (pxy *BaseProxy) Context() context.Context {
	return pxy.ctx
}

func (pxy *BaseProxy) GetUsedPortsNum() int {
	return pxy.usedPortsNum
}

func (pxy *BaseProxy) GetResourceController() *controller.ResourceController {
	return pxy.rc
}

func (pxy *BaseProxy) GetUserInfo() auth.UserInfo {
	return pxy.userInfo
}

func (pxy *BaseProxy) GetLoginMsg() *msg.Login {
	return pxy.loginMsg
}

func (pxy *BaseProxy) GetConfigurer() v1.ProxyConfigurer {
	return pxy.configurer
}

func (pxy *BaseProxy) Close() {
	xl := xlog.FromContextSafe(pxy.ctx)
	xl.Infof("proxy closing")
}

func (pxy *BaseProxy) GetWorkConnFromPool(src, dst net.Addr) (workConn net.Conn, err error) {
	xl := xlog.FromContextSafe(pxy.ctx)
	for i := 0; i < pxy.poolCount+1; i++ {
		var pxyWorkConn *WorkConn
		if pxyWorkConn, err = pxy.getWorkConnFn(); err != nil {
			xl.Warnf("failed to get work connection: %v", err)
			return
		}
		xl.Debugf("get a new work connection: [%s]", pxyWorkConn.conn.RemoteAddr().String())
		xl.Spawn().AppendPrefix(pxy.GetName())

		var (
			srcAddr    string
			dstAddr    string
			srcPortStr string
			dstPortStr string
			srcPort    uint64
			dstPort    uint64
		)

		if src != nil {
			srcAddr, srcPortStr, _ = net.SplitHostPort(src.String())
			srcPort, _ = strconv.ParseUint(srcPortStr, 10, 16)
		}
		if dst != nil {
			dstAddr, dstPortStr, _ = net.SplitHostPort(dst.String())
			dstPort, _ = strconv.ParseUint(dstPortStr, 10, 16)
		}
		workConn, err = pxyWorkConn.Start(&msg.StartWorkConn{
			ProxyName: pxy.GetName(),
			SrcAddr:   srcAddr,
			SrcPort:   uint16(srcPort),
			DstAddr:   dstAddr,
			DstPort:   uint16(dstPort),
			Error:     "",
		})
		if err != nil {
			xl.Warnf("failed to send message to work connection from pool: %v, times: %d", err, i)
			pxyWorkConn.Close()
			workConn = nil
		} else {
			workConn = netpkg.NewContextConn(pxy.ctx, workConn)
			break
		}
	}

	if err != nil {
		xl.Errorf("try to get work connection failed in the end")
		return
	}
	return
}

type Options struct {
	UserInfo           auth.UserInfo
	LoginMsg           *msg.Login
	PoolCount          int
	ResourceController *controller.ResourceController
	GetWorkConnFn      GetWorkConnFn
	Configurer         v1.ProxyConfigurer
	ServerCfg          *v1.ServerConfig
	EncryptionKey      []byte
	WireProtocol       string
	UDPPacketCodec     string
}

func NewProxy(ctx context.Context, options *Options) (pxy Proxy, err error) {
	configurer := options.Configurer
	xl := xlog.FromContextSafe(ctx).Spawn().AppendPrefix(configurer.GetBaseConfig().Name)

	basePxy := BaseProxy{
		name:          configurer.GetBaseConfig().Name,
		rc:            options.ResourceController,
		poolCount:     options.PoolCount,
		getWorkConnFn: options.GetWorkConnFn,
		serverCfg:     options.ServerCfg,
		encryptionKey: options.EncryptionKey,
		xl:            xl,
		ctx:           xlog.NewContext(ctx, xl),
		userInfo:      options.UserInfo,
		loginMsg:      options.LoginMsg,
		configurer:    configurer,
		wireProtocol:  options.WireProtocol,
	}

	factory := proxyFactoryRegistry[reflect.TypeOf(configurer)]
	if factory == nil {
		return pxy, fmt.Errorf("proxy type not support")
	}
	pxy = factory(&basePxy)
	if pxy == nil {
		return nil, fmt.Errorf("proxy not created")
	}
	return pxy, nil
}

type Manager struct {
	pxys map[string]Proxy
	mu   sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		pxys: make(map[string]Proxy),
	}
}

func (pm *Manager) Add(name string, pxy Proxy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.pxys[name]; ok {
		return fmt.Errorf("proxy name [%s] is already in use", name)
	}
	pm.pxys[name] = pxy
	return nil
}

func (pm *Manager) Exist(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, ok := pm.pxys[name]
	return ok
}

func (pm *Manager) Del(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.pxys, name)
}

func (pm *Manager) GetByName(name string) (pxy Proxy, ok bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pxy, ok = pm.pxys[name]
	return
}
