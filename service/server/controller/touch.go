package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/iopq/xraya/common"
	"github.com/iopq/xraya/core/touch"
	"github.com/iopq/xraya/core/v2ray"
	"github.com/iopq/xraya/db/configure"
	"github.com/iopq/xraya/server/service"
)

func GetTouch(ctx *gin.Context) {
	updatingMu.Lock()
	if updating {
		common.ResponseError(ctx, processingErr)
		updatingMu.Unlock()
		return
	}
	updatingMu.Unlock()
	defer func() {
		updatingMu.Lock()
		updatingMu.Unlock()
	}()
	getTouch(ctx)

}
func getTouch(ctx *gin.Context) {
	running := v2ray.ProcessManager.Running()
	t := touch.GenerateTouch()
	common.ResponseSuccess(ctx, gin.H{
		"running": running,
		"touch":   t,
	})
}

func DeleteTouch(ctx *gin.Context) {
	updatingMu.Lock()
	if updating {
		common.ResponseError(ctx, processingErr)
		updatingMu.Unlock()
		return
	}
	updating = true
	updatingMu.Unlock()
	defer func() {
		updatingMu.Lock()
		updating = false
		updatingMu.Unlock()
	}()

	var ws configure.Whiches
	err := ctx.ShouldBindJSON(&ws)
	if err != nil {
		common.ResponseError(ctx, logError("bad request"))
		return
	}
	err = service.DeleteWhich(ws.Get())
	if err != nil {
		common.ResponseError(ctx, logError(err))
		return
	}
	getTouch(ctx)
}
