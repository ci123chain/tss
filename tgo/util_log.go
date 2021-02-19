package tgo

func UtilLogError(msg string) {
	LogErrorf(LogNameDefault, msg)
}

func UtilLogErrorf(format string, a ...interface{}) {
	LogErrorf(LogNameDefault, format, a...)
}

func UtilLogInfo(msg string) {
	LogInfof(LogNameDefault, msg)
}

func UtilLogInfof(format string, a ...interface{}) {
	LogInfof(LogNameDefault, format, a...)
}

func UtilLogDebug(msg string) {
	LogDebugf(LogNameDefault, msg)
}

func UtilLogDebugf(format string, a ...interface{}) {
	LogDebugf(LogNameDefault, format, a...)
}

func NewUtilLog() *Log {
	return &Log{}
}

func (l *Log) Error(format string, a ...interface{}) {
	UtilLogErrorf(format, a...)
}

func (l *Log) Info(format string, a ...interface{}) {
	UtilLogInfof(format, a...)
}
