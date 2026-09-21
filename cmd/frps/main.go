package main

import (
	"github.com/fatedier/frp/pkg/util/system"
)

func main() {
	system.EnableCompatibilityMode()
	Execute()
}
