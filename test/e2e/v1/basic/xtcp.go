package basic

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/stunserver"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"
)

var _ = ginkgo.Describe("[Feature: XTCP]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("P2P over local assisted addresses", func() {
		stun, err := stunserver.New()
		framework.ExpectNoError(err)
		stun.Run()
		defer stun.Close()

		serverConf := consts.DefaultServerConfig
		bindPortName := port.GenName("XTCP")
		stunAddr := stun.Addr()

		serverClientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			natHoleStunServer = "%s"
			[[proxies]]
			name = "foo"
			type = "xtcp"
			secretKey = "abcdefg"
			localPort = {{ .%s }}
			allowUsers = ["*"]
			`, stunAddr, framework.TCPEchoServerPort)

		visitorClientConf := consts.DefaultClientConfig + fmt.Sprintf(`
			natHoleStunServer = "%s"
			[[visitors]]
			name = "foo-visitor"
			type = "xtcp"
			serverName = "foo"
			secretKey = "abcdefg"
			bindPort = {{ .%s }}
			keepTunnelOpen = true
			protocol = "quic"
			`, stunAddr, bindPortName)

		f.RunProcesses(serverConf, []string{serverClientConf, visitorClientConf})
		framework.NewRequestExpect(f).
			RequestModify(func(r *request.Request) {
				r.Timeout(25 * time.Second)
			}).
			PortName(bindPortName).
			Ensure()
	})
})
