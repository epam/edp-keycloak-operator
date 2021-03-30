package mock

import "github.com/go-logr/logr"

type InfoLogger struct {
}

func (InfoLogger) Info(msg string, keysAndValues ...interface{}) {

}

func (InfoLogger) Enabled() bool {
	return true
}

type Logger struct {
	InfoLogger
	errors []error
}

func (l *Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.errors = append(l.errors, err)
}

func (l *Logger) LastError() error {
	if len(l.errors) == 0 {
		return nil
	}

	return l.errors[len(l.errors)-1]
}

func (Logger) V(level int) logr.InfoLogger {
	return InfoLogger{}
}

func (l *Logger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return l
}

func (l *Logger) WithName(name string) logr.Logger {
	return l
}
