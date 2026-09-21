package framework

import (
	"github.com/fatedier/frp/test/e2e/mock/server"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

const (
	TCPEchoServerPort = "TCPEchoServerPort"
)

type MockServers struct {
	tcpEchoServer server.Server
}

func NewMockServers(portAllocator *port.Allocator) *MockServers {
	s := &MockServers{}
	tcpPort := portAllocator.Get()
	s.tcpEchoServer = streamserver.New(streamserver.TCP, streamserver.WithBindPort(tcpPort))
	return s
}

func (m *MockServers) Run() error {
	return m.tcpEchoServer.Run()
}

func (m *MockServers) Close() {
	m.tcpEchoServer.Close()
}

func (m *MockServers) GetTemplateParams() map[string]any {
	return map[string]any{
		TCPEchoServerPort: m.tcpEchoServer.BindPort(),
	}
}
