package consts

import (
	"fmt"
	"time"

	"github.com/fatedier/frp/test/e2e/pkg/port"
)

const (
	TestString = "frp is a fast reverse proxy to help you expose a local server behind a NAT or firewall to the internet."

	DefaultTimeout = 2 * time.Second
)

var (
	PortServerName string

	DefaultServerConfig = `
bindPort = {{ .%s }}
log.level = "trace"
`

	DefaultClientConfig = `
serverAddr = "127.0.0.1"
serverPort = {{ .%s }}
loginFailExit = false
log.level = "trace"
`
)

func init() {
	PortServerName = port.GenName("Server")
	DefaultServerConfig = fmt.Sprintf(DefaultServerConfig, port.GenName("Server"))
	DefaultClientConfig = fmt.Sprintf(DefaultClientConfig, port.GenName("Server"))
}
