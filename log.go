package gocache

type Logger interface {
	Debug(f string, args ...interface{})
	Info(f string, args ...interface{})
	Error(f string, args ...interface{})
}

type NopLogger struct {
}

func (l *NopLogger) Debug(f string, args ...interface{}) {}

func (l *NopLogger) Info(f string, args ...interface{}) {}

func (l *NopLogger) Error(f string, args ...interface{}) {}
