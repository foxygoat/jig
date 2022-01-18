package serve

import (
	"fmt"
	"io"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		panic(l)
	}
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	switch string(text) {
	case "debug":
		*l = LogLevelDebug
	case "info":
		*l = LogLevelInfo
	case "warn", "warning":
		*l = LogLevelWarn
	case "error":
		*l = LogLevelError
	default:
		return fmt.Errorf("invalid log level %q", text)
	}
	return nil
}

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type LoggerFunc func(level LogLevel, format string, args ...interface{})

func (l LoggerFunc) Debugf(format string, args ...interface{}) { l(LogLevelDebug, format, args...) }
func (l LoggerFunc) Infof(format string, args ...interface{})  { l(LogLevelInfo, format, args...) }
func (l LoggerFunc) Warnf(format string, args ...interface{})  { l(LogLevelWarn, format, args...) }
func (l LoggerFunc) Errorf(format string, args ...interface{}) { l(LogLevelError, format, args...) }

func NewLogger(w io.Writer, minLevel LogLevel) Logger {
	return LoggerFunc(func(level LogLevel, format string, args ...interface{}) {
		if level >= minLevel {
			fmt.Fprintf(w, level.String()+": "+format+"\n", args...)
		}
	})
}
