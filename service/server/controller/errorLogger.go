package controller

import (
	"fmt"
	"github.com/iopq/xraya/pkg/util/log"
)

func logError(err interface{}) error {
	e := fmt.Errorf("%v", err)
	log.Error("%v", e)
	return e
}
