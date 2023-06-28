package ulog

import (
	"fmt"
	"os"
)

var (
	globalFormat = NewDefaultFormat(func(s string) {
		os.Stdout.WriteString(s)
	})
	globalLogger = NewLogger(globalFormat)
)

func GlobalLogger() *Logger {
	return globalLogger
}

func GlobalFormat() *DefaultFormat {
	return globalFormat
}

func Info(format string, args ...interface{}) {
	globalLogger.Log(LevelInfo, defaultCaller, format, args...)
}

func Debug(format string, args ...interface{}) {
	globalLogger.Log(LevelDebug, defaultCaller, format, args...)
}

func Warn(format string, args ...interface{}) {
	globalLogger.Log(LevelWarn, defaultCaller, format, args...)
}

func Error(format string, args ...interface{}) {
	globalLogger.Log(LevelError, defaultCaller, format+"\r\nError Stack:\r\n%s",
		append(args, Stack(1000, 1))...)
}

func Fatal(format string, args ...interface{}) {
	globalLogger.Log(LevelFatal, defaultCaller, format, args...)
	panic(fmt.Sprintf(format, args...))
}

func Printf(format string, args ...interface{}) {
	globalLogger.Log(LevelPrintf, defaultCaller, format, args...)
}

func Timer() *TimerLogger {
	return globalLogger.Timer()
}

func Progress(length int, total float64, unit string) *ProgressLogger {
	return globalLogger.Progress(length, total, unit)
}

func Register(format Format) {
	globalLogger.Register(format)
}
