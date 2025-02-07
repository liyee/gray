package glog

var StdZinxLog = NewGtcpLog("", BitDefault)

// Flags gets the flags of StdZinxLog
func Flags() int {
	return StdZinxLog.Flags()
}

// ResetFlags sets the flags of StdZinxLog
func ResetFlags(flag int) {
	StdZinxLog.ResetFlags(flag)
}

// AddFlag adds a flag to StdZinxLog
func AddFlag(flag int) {
	StdZinxLog.AddFlag(flag)
}

// SetPrefix sets the log prefix of StdZinxLog
func SetPrefix(prefix string) {
	StdZinxLog.SetPrefix(prefix)
}

// SetLogFile sets the log file of StdZinxLog
func SetLogFile(fileDir string, fileName string) {
	StdZinxLog.SetLogFile(fileDir, fileName)
}

// SetMaxAge 最大保留天数
func SetMaxAge(ma int) {
	StdZinxLog.SetMaxAge(ma)
}

// SetMaxSize 单个日志最大容量 单位：字节
func SetMaxSize(ms int64) {
	StdZinxLog.SetMaxSize(ms)
}

// SetCons 同时输出控制台
func SetCons(b bool) {
	StdZinxLog.SetCons(b)
}

// SetLogLevel sets the log level of StdZinxLog
func SetLogLevel(logLevel int) {
	StdZinxLog.SetLogLevel(logLevel)
}

func Debugf(format string, v ...interface{}) {
	StdZinxLog.Debugf(format, v...)
}

func Debug(v ...interface{}) {
	StdZinxLog.Debug(v...)
}

func Infof(format string, v ...interface{}) {
	StdZinxLog.Infof(format, v...)
}

func Info(v ...interface{}) {
	StdZinxLog.Info(v...)
}

func Warnf(format string, v ...interface{}) {
	StdZinxLog.Warnf(format, v...)
}

func Warn(v ...interface{}) {
	StdZinxLog.Warn(v...)
}

func Errorf(format string, v ...interface{}) {
	StdZinxLog.Errorf(format, v...)
}

func Error(v ...interface{}) {
	StdZinxLog.Error(v...)
}

func Fatalf(format string, v ...interface{}) {
	StdZinxLog.Fatalf(format, v...)
}

func Fatal(v ...interface{}) {
	StdZinxLog.Fatal(v...)
}

func Panicf(format string, v ...interface{}) {
	StdZinxLog.Panicf(format, v...)
}

func Panic(v ...interface{}) {
	StdZinxLog.Panic(v...)
}

func Stack(v ...interface{}) {
	StdZinxLog.Stack(v...)
}

func init() {
	// Since the StdZinxLog object wraps all output methods with an extra layer, the call depth is one more than a normal logger object
	// The call depth of a regular zinxLogger object is 2, and the call depth of StdZinxLog is 3
	// (因为StdZinxLog对象 对所有输出方法做了一层包裹，所以在打印调用函数的时候，比正常的logger对象多一层调用
	// 一般的zinxLogger对象 calldDepth=2, StdZinxLog的calldDepth=3)
	StdZinxLog.calldDepth = 3
}
