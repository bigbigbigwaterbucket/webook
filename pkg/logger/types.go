package logger

type Logger interface {
	Info(msg string, args ...Filed)
	Debug(msg string, args ...Filed)
	Warn(msg string, args ...Filed)
	Error(msg string, args ...Filed)
}

type Filed struct {
	Key   string
	Value any
}
