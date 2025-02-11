package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/iopq/xraya/common"
	"github.com/iopq/xraya/conf"
	"github.com/iopq/xraya/core/v2ray/asset/dat"
	"github.com/iopq/xraya/core/v2ray/service"
	"github.com/iopq/xraya/core/v2ray/where"
	"net/http"
)

func GetVersion(ctx *gin.Context) {
	var dohValid string
	var vlessValid int
	var lite int

	variant, ver, err := where.GetV2rayServiceVersion()
	if err == nil && variant == where.V2ray {
		if ok, _ := common.VersionGreaterEqual(ver, "4.27.0"); ok {
			// 1: vless
			vlessValid++
			if ok, _ = common.VersionGreaterEqual(ver, "4.30.0"); ok {
				// 2: xtls-rprx-origin
				vlessValid++
				if ok, _ = common.VersionGreaterEqual(ver, "4.31.0"); ok {
					// 3: xtls-rprx-direct, xtls-rprx-direct-udp443
					vlessValid++
				}
			}
		}
		err = service.CheckDohSupported()
		if err == nil {
			dohValid = "yes"
		} else {
			dohValid = err.Error()
		}
	} else {
		vlessValid = 3
		dohValid = "yes"
	}
	if conf.GetEnvironmentConfig().Lite {
		lite = 1
	}
	common.ResponseSuccess(ctx, gin.H{
		"version":          conf.Version,
		"foundNew":         conf.FoundNew,
		"remoteVersion":    conf.RemoteVersion,
		"serviceValid":     service.IsV2rayServiceValid(),
		"dohValid":         dohValid,
		"vlessValid":       vlessValid,
		"lite":             lite,
		"loadBalanceValid": service.CheckObservatorySupported() == nil,
	})
}

func GetRemoteGFWListVersion(ctx *gin.Context) {
	//c, err := httpClient.GetHttpClientAutomatically()
	//if err != nil {
	//	tools.ResponseError(ctx, err)
	//	return
	//}
	g, err := dat.GetRemoteGFWListUpdateTime(http.DefaultClient)
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	common.ResponseSuccess(ctx, gin.H{"remoteGFWListVersion": g.UpdateTime.Local().Format("2006-01-02")})
}
