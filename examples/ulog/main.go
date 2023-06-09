package main

import (
	"fmt"
	"time"

	"uw/ulog"
)

func main() {
	t := ulog.Timer()

	for i := 0; i < 1000; i++ {
		ulog.Info("hello world %d", i)
	}

	t.End("write 1000 logs")

	p := ulog.Progress(10, 100, "")
	fmt.Print("\r\n")
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 100)
		fmt.Printf("\033[1A\033[K")
		p.Append(1, "hello world")
	}
}
