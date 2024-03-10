package ulog

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Level uint8

const (
	LevelMuted  Level = 0           // 关闭输出
	LevelPrintf Level = (1 << iota) // 格式化输出
	LevelTrace                      // 追溯信息
	LevelDebug                      // 调试信息
	LevelInfo                       // 普通信息
	LevelWarn                       // 警告消息
	LevelError                      // 错误消息
	LevelFatal                      // 致命错误

	defaultCaller int = 2 // 默认追踪调用层级
)

type Logger struct {
	logPool    sync.Pool
	formatList []Format
}

type Log struct {
	Level   Level     // 等级
	File    string    // 追溯文件
	Line    int       // 追溯行号
	Message string    // 消息
	Time    time.Time // 时间
}

type Format interface {
	Write(log *Log)
}

func LevelName(level Level) string {
	switch level {
	case LevelMuted:
		return "MUTED"
	case LevelTrace:
		return "TRACE"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelDebug:
		return "DEBUG"
	case LevelFatal:
		return "FATAL"
	case LevelPrintf:
		return "PRINTF"
	default:
		return "UNKNOWN"
	}
}

func NewLogger(formatList ...Format) *Logger {
	l := &Logger{
		logPool: sync.Pool{
			New: func() interface{} {
				return &Log{}
			},
		},
		formatList: formatList,
	}

	return l
}

func (l *Logger) Writer(g *Log) {
	defer l.logPool.Put(g)
	for i := 0; i < len(l.formatList); i++ {
		l.formatList[i].Write(g)
	}
}

func (l *Logger) Register(format Format) {
	l.formatList = append(l.formatList, format)
}

func (l *Logger) Unregister() {
	l.formatList = []Format{}
}

func (l *Logger) Log(level Level, skipCaller int, format string, args ...interface{}) {
	g := l.logPool.Get().(*Log)
	g.Level = level
	g.Message = fmt.Sprintf(format, args...)
	_, g.File, g.Line, _ = runtime.Caller(skipCaller)
	g.Time = time.Now()

	l.Writer(g)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.Log(LevelPrintf, defaultCaller, format, args...)
}

func (l *Logger) Trace(format string, args ...interface{}) {
	l.Log(LevelTrace, defaultCaller, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(LevelDebug, defaultCaller, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(LevelInfo, defaultCaller, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.Log(LevelWarn, defaultCaller, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(LevelError, defaultCaller, format+"\r\nError Stack:\r\n%s",
		append(args, Stack(1000, 1))...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.Log(LevelFatal, defaultCaller, format, args...)
	panic(fmt.Sprintf(format, args...))
}
