package ulog

import (
	"sync"
	"testing"
)

func TestLogger_Log(t *testing.T) {
	t.Run("QAQ", func(t *testing.T) {
		l := &Logger{
			logPool: sync.Pool{
				New: func() interface{} {
					return &Log{}
				},
			},
			formatList: []Format{},
		}
		l.Log(LevelDebug, 1, "Test")
	})
}

func BenchmarkLog(b *testing.B) {
	l := &Logger{
		logPool: sync.Pool{
			New: func() interface{} {
				return &Log{}
			},
		},
		formatList: []Format{},
	}
	for i := 0; i < b.N; i++ {
		l.Log(LevelDebug, 1, "Test")
	}
}
