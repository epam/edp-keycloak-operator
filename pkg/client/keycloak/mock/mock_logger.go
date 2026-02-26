package mock

import "github.com/go-logr/logr"

func NewLogr() logr.Logger {
	return logr.New(&Logger{})
}

type Logger struct {
	errors       []error
	infoMessages map[string][]any
}

// Init implements logr.LogSink.
func (log *Logger) Init(logr.RuntimeInfo) {
}

// Info implements logr.InfoLogger.
func (l *Logger) Info(level int, msg string, keysAndValues ...any) {
	if l.infoMessages == nil {
		l.infoMessages = make(map[string][]any)
	}

	l.infoMessages[msg] = keysAndValues
}

func (l *Logger) InfoMessages() map[string][]any {
	return l.infoMessages
}

// Enabled implements logr.InfoLogger.
func (Logger) Enabled(level int) bool {
	return true
}

func (l *Logger) Error(err error, msg string, keysAndValues ...any) {
	l.errors = append(l.errors, err)
}

func (l Logger) LastError() error {
	if len(l.errors) == 0 {
		return nil
	}

	return l.errors[len(l.errors)-1]
}

// WithName implements logr.Logger.
func (log *Logger) WithName(_ string) logr.LogSink {
	return log
}

// WithValues implements logr.Logger.
func (log *Logger) WithValues(_ ...any) logr.LogSink {
	return log
}
