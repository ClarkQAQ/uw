package ulog

import (
	"fmt"
	"path"
	"time"
)

var (
	LevelColor   Level = 128
	DefaultLevel       = LevelDebug | LevelInfo | LevelWarn | LevelError |
		LevelFatal | LevelPrintf | LevelColor
	DefaultLevelNoColor = DefaultLevel ^ LevelColor
)

type DefaultFormat struct {
	location *time.Location
	writer   func(s string)
	level    Level
}

func NewDefaultFormat(f func(s string)) *DefaultFormat {
	return &DefaultFormat{
		writer:   f,
		location: time.Local,
		level:    DefaultLevel,
	}
}

func (sw *DefaultFormat) SetWriter(f func(s string)) {
	sw.writer = f
}

func (sw *DefaultFormat) SetLocation(location *time.Location) {
	sw.location = location
}

func (sw *DefaultFormat) GetLevel() Level {
	return sw.level
}

func (sw *DefaultFormat) SetLevel(val Level) {
	sw.level = val
}

func (sw *DefaultFormat) Write(log *Log) {
	if sw.level&log.Level != 0 || LevelFatal&log.Level != 0 {
		sw.writer(sw.Format(log))
	}
}

func (sw *DefaultFormat) Format(log *Log) string {
	if sw.level&LevelColor != 0 {
		return sw.PrettyFormat(log)
	}

	return sw.PureFormat(log)
}

func (sw *DefaultFormat) PureFormat(log *Log) string {
	return fmt.Sprintf("%s %s %s:%d %s\r\n",
		LevelName(log.Level),
		log.Time.In(sw.location).Format("06-01-02 15:04:05.000"),
		path.Base(log.File), log.Line,
		fmt.Sprintf(log.Format, log.Args...),
	)
}

func (sw *DefaultFormat) PrettyFormat(log *Log) string {
	return fmt.Sprintf("%s %s %s %s\r\n",
		levelPretty(log.Level),
		SetANSI(ANSI.Grey, log.Time.In(sw.location).Format("06-01-02 15:04:05.000")),
		SetANSI(ANSI.Magenta, fmt.Sprintf("%s:%d", path.Base(log.File), log.Line)),
		fmt.Sprintf(log.Format, log.Args...),
	)
}

func levelPretty(level Level) string {
	switch level {
	case LevelInfo:
		return SetANSI(ANSI.Green, SetANSI(ANSI.Bold, LevelName(level)))
	case LevelWarn:
		return SetANSI(ANSI.Yellow, SetANSI(ANSI.Bold, LevelName(level)))
	case LevelError:
		return SetANSI(ANSI.Red, SetANSI(ANSI.Bold, LevelName(level)))
	case LevelDebug:
		return SetANSI(ANSI.Blue, SetANSI(ANSI.Bold, LevelName(level)))
	case LevelFatal:
		return SetANSI(ANSI.Red, SetANSI(ANSI.Flash, SetANSI(ANSI.Bold, LevelName(level))))
	case LevelPrintf:
		return SetANSI(ANSI.Blue, SetANSI(ANSI.Bold, LevelName(level)))
	default:
		return SetANSI(ANSI.White, SetANSI(ANSI.Bold, LevelName(level)))
	}
}
