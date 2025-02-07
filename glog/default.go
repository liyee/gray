package glog

import (
	"context"
	"fmt"

	"github.com/liyee/gtcp/giface"
)

var gLogInstance giface.ILogger = new(gtcpDefaultLog)

type gtcpDefaultLog struct{}

func (log *gtcpDefaultLog) InfoF(format string, v ...interface{}) {
	StdZinxLog.Infof(format, v...)
}

func (log *gtcpDefaultLog) ErrorF(format string, v ...interface{}) {
	StdZinxLog.Errorf(format, v...)
}

func (log *gtcpDefaultLog) DebugF(format string, v ...interface{}) {
	StdZinxLog.Debugf(format, v...)
}

func (log *gtcpDefaultLog) InfoFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdZinxLog.Infof(format, v...)
}

func (log *gtcpDefaultLog) ErrorFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdZinxLog.Errorf(format, v...)
}

func (log *gtcpDefaultLog) DebugFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdZinxLog.Debugf(format, v...)
}

func SetLogger(newlog giface.ILogger) {
	gLogInstance = newlog
}

func Ins() giface.ILogger {
	return gLogInstance
}
