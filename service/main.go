package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/iopq/xraya/conf/report"
	_ "github.com/iopq/xraya/pkg/plugin/pingtunnel"
	_ "github.com/iopq/xraya/pkg/plugin/simpleobfs"
	_ "github.com/iopq/xraya/pkg/plugin/socks5"
	_ "github.com/iopq/xraya/pkg/plugin/ss"
	_ "github.com/iopq/xraya/pkg/plugin/ssr"
	_ "github.com/iopq/xraya/pkg/plugin/tcp"
	_ "github.com/iopq/xraya/pkg/plugin/tls"
	_ "github.com/iopq/xraya/pkg/plugin/trojanc"
	_ "github.com/iopq/xraya/pkg/plugin/ws"
	"github.com/iopq/xraya/pkg/util/log"
	"runtime"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	checkEnvironment()
	if runtime.GOOS == "linux" {
		checkTProxySupportability()
	}
	initConfigure()
	checkUpdate()
	hello()
	if err := run(); err != nil {
		log.Fatal("main: %v", err)
	}
}
