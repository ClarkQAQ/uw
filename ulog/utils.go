package ulog

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
)

// 终端颜色数据
var ANSI = struct {
	Grey      string // 灰色前景
	Red       string // 红色前景
	Green     string // 绿色前景
	Yellow    string // 黄色前景
	Blue      string // 蓝色前景
	Magenta   string // 品红前景
	Cyan      string // 青色前景
	White     string // 白色前景
	Bold      string // 粗体
	Flash     string // 闪烁
	ArrowUp   string // 上移
	ClearLine string // 清除行
	Reset     string // 重置颜色
}{
	"\033[90m",
	"\033[31m",
	"\033[32m",
	"\033[33m",
	"\033[34m",
	"\033[35m",
	"\033[36m",
	"\033[37m",
	"\033[1m",
	"\033[5m",
	"\033[%dA", // %d: 行数, 用于 fmt 格式化
	"\033[K",
	"\033[0m",
}

var ANSIRegexp = regexp.MustCompile(`\033\[[0-9;]+m`) // 匹配 ANSI 字符

func SetANSI(ansi string, val string) string {
	return ansi + val + ANSI.Reset
}

func CleanANSI(s string) string {
	return ANSIRegexp.ReplaceAllString(s, "")
}

func SprintfANSI(ansi string, format string, val ...interface{}) string {
	return ansi + fmt.Sprintf(format, val...) + ANSI.Reset
}

func Stack(maxDepth, skip int) string {
	buffer, index := bytes.NewBuffer(nil), 0
	for i := skip; i < maxDepth; i++ {
		if pc, file, line, ok := runtime.Caller(i); ok {
			index++

			if fn := runtime.FuncForPC(pc); fn != nil {
				buffer.WriteString(fmt.Sprintf("%d.%s\n\t%s:%d\n",
					index, fn.Name(), file, line))
				continue
			}

			buffer.WriteString(fmt.Sprintf("%d.unknow\n\t%s:%d\n",
				index, file, line))
		} else {
			break
		}
	}
	return buffer.String()
}
