package loggerx

type Logger interface {
	Debug(msg string, args ...Field)
	Info(msg string, args ...Field)
	Warn(msg string, args ...Field)
	Error(msg string, args ...Field)
}

type Field struct { // 适配器模式（不同接口，装饰器模式：相同接口）
	Key   string
	Value any
}
