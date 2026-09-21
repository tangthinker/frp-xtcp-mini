package main

import (
	"github.com/fatedier/frp/cmd/frpc/sub"
	"github.com/fatedier/frp/pkg/util/system"
)

func main() {
	system.EnableCompatibilityMode()
	sub.Execute()
}
