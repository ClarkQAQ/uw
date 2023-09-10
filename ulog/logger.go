package ulog

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Level uint8

const (
	LevelMuted  Level = 0           // 关闭输出
	LevelPrintf Level = (1 << iota) // 格式化输出
	LevelDebug                      // 调试信息
	LevelInfo                       // 普通信息
	LevelWarn                       // 警告消息
	LevelError                      // 错误消息
	LevelFatal                      // 致命错误

	defaultCaller int = 2 // 默认追踪调用层级
)

type Logger struct {
	cache      chan *Log
	asyncOnce  *sync.Once
	formatList []Format
}

type Log struct {
	Level  Level         // 等级
	File   string        // 追溯文件
	Line   int           // 追溯行号
	Format string        // 消息
	Args   []interface{} // 消息参数
	Time   time.Time     // 时间
}

type Format interface {
	Write(log *Log)
}

func LevelName(level Level) string {
	switch level {
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
		asyncOnce:  &sync.Once{},
		formatList: formatList,
	}

	return l
}

func (l *Logger) Async(worker int, cacheSize int64) error {
	if l.cache != nil {
		return fmt.Errorf("logger already async")
	}

	if worker < 1 {
		return fmt.Errorf("invalid worker")
	}

	l.cache = make(chan *Log, cacheSize)
	l.asyncLogger(worker)
	return nil
}

// 该方法会导致关闭前的一段日志顺序不一致
func (l *Logger) CloseWait(ctx context.Context) {
	if l.cache != nil {
		for {
			select {
			case <-ctx.Done():
			case g := <-l.cache:
				l.cache <- g
				continue
			default:
			}

			l.Close()
			break
		}
	}
}

func (l *Logger) Close() {
	defer func() { _ = recover() }()
	close(l.cache)
}

func (l *Logger) asyncLogger(worker int) {
	l.asyncOnce.Do(func() {
		for i := 0; i < worker; i++ {
			go func(l *Logger) {
				for g := range l.cache {
					l.Writer(g)
				}
			}(l)
		}
	})
}

func (l *Logger) Writer(g *Log) {
	for i := 0; i < len(l.formatList); i++ {
		l.formatList[i].Write(g)
	}
}

func (l *Logger) Register(format Format) {
	l.formatList = append(l.formatList, format)
}

func (l *Logger) UnregisterAll() {
	l.formatList = []Format{}
}

func (l *Logger) Log(level Level, skipCaller int, format string, args ...interface{}) {
	g := &Log{
		Level:  level,
		Time:   time.Now(),
		Format: format,
		Args:   args,
	}

	_, g.File, g.Line, _ = runtime.Caller(skipCaller)

	if l.cache != nil {
		l.cache <- g
		return
	}

	l.Writer(g)
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

func (l *Logger) Printf(format string, args ...interface{}) {
	l.Log(LevelPrintf, defaultCaller, format, args...)
}
