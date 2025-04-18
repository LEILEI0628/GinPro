package loggerx

// NoneLogger 无日志打印，用于调试
type NoneLogger struct {
}

func (n *NoneLogger) Debug(msg string, args ...Field) {
}

func (n *NoneLogger) Info(msg string, args ...Field) {
}

func (n *NoneLogger) Warn(msg string, args ...Field) {
}

func (n *NoneLogger) Error(msg string, args ...Field) {
}
