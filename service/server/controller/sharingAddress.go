package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/json-iterator/go"
	"github.com/iopq/xraya/common"
	"github.com/iopq/xraya/db/configure"
	"github.com/iopq/xraya/server/service"
)

func GetSharingAddress(ctx *gin.Context) {
	var w configure.Which
	err := jsoniter.Unmarshal([]byte(ctx.Query("touch")), &w)
	if err != nil {
		common.ResponseError(ctx, logError("bad request"))
		return
	}
	addr, err := service.GetSharingAddress(&w)
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	common.ResponseSuccess(ctx, gin.H{"sharingAddress": addr})
}
