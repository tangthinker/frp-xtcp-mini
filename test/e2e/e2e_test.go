package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"

	_ "github.com/onsi/ginkgo/v2"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework"
	_ "github.com/fatedier/frp/test/e2e/v1/basic"
)

func handleFlags() {
	framework.RegisterCommonFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	handleFlags()

	if err := framework.ValidateTestContext(&framework.TestContext); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.InitLogger("console", framework.TestContext.LogLevel, 0, true)
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
