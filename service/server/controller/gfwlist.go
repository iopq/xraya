package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/iopq/xraya/common"
	"github.com/iopq/xraya/core/v2ray/asset/dat"
)

func PutGFWList(ctx *gin.Context) {
	localGFWListVersion, err := dat.CheckAndUpdateGFWList()
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	common.ResponseSuccess(ctx, gin.H{
		"localGFWListVersion": localGFWListVersion,
	})
}
