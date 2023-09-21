package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/iopq/xraya/common"
	"github.com/iopq/xraya/db/configure"
	"github.com/iopq/xraya/server/service"
)

func PostImport(ctx *gin.Context) {
	var data struct {
		URL   string `json:"url"`
		Which *configure.Which
	}
	err := ctx.ShouldBindJSON(&data)
	if err != nil {
		common.ResponseError(ctx, logError("bad request"))
		return
	}
	err = service.Import(data.URL, data.Which)
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	getTouch(ctx)
}
