package glog

import (
	"context"
	"fmt"

	"github.com/liyee/gray/giface"
)

var gLogInstance giface.ILogger = new(grayDefaultLog)

type grayDefaultLog struct{}

func (log *grayDefaultLog) InfoF(format string, v ...interface{}) {
	StdZinxLog.Infof(format, v...)
}

func (log *grayDefaultLog) ErrorF(format string, v ...interface{}) {
	StdZinxLog.Errorf(format, v...)
}

func (log *grayDefaultLog) DebugF(format string, v ...interface{}) {
	StdZinxLog.Debugf(format, v...)
}

func (log *grayDefaultLog) InfoFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdZinxLog.Infof(format, v...)
}

func (log *grayDefaultLog) ErrorFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdZinxLog.Errorf(format, v...)
}

func (log *grayDefaultLog) DebugFX(ctx context.Context, format string, v ...interface{}) {
	fmt.Println(ctx)
	StdZinxLog.Debugf(format, v...)
}

func SetLogger(newlog giface.ILogger) {
	gLogInstance = newlog
}

func Ins() giface.ILogger {
	return gLogInstance
}
