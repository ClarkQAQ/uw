package main

import (
	"context"
	"os"
	"time"

	"uw/uboot"
	"uw/ulog"
)

type CustomWriter func(p []byte) (n int, err error)

func (w CustomWriter) Write(p []byte) (n int, err error) {
	return w(p)
}

func main() {
	f, e := os.OpenFile("test.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if e != nil {
		ulog.Fatal("open log file error: %s", e)
	}

	ulog.GlobalFormat().SetWriter(func(s string) {
		os.Stdout.WriteString(s)
		f.Write(ulog.ANSIRegexp.ReplaceAll([]byte(s), nil))
	})

	uboot.NewBoot().Register(
		uboot.Uint("normal_1", uboot.UintNormal, func(c *uboot.Context) error {
			c.Printf("normal_1")

			// time.Sleep(1000 * time.Millisecond)
			c.Printf("normal_1 done")
			return nil
		}).Timeout(500*time.Millisecond),
		uboot.Uint("normal_2", uboot.UintNormal, func(c *uboot.Context) error {
			c.Printf("normal_2")
			time.Sleep(1000 * time.Millisecond)
			c.Require(context.Background(), "normal_1")
			c.Printf("normal_2 done")
			return nil
		}),
		uboot.Uint("background", uboot.UintBackground, func(c *uboot.Context) error {
			c.Printf("background")
			time.Sleep(10 * time.Second)
			c.Printf("background done")
			return nil
		}),
		uboot.Uint("after_1", uboot.UintAfter, func(c *uboot.Context) error {
			c.Printf("after_1")
			time.Sleep(500 * time.Millisecond)
			c.Printf("after_1 done")
			return nil
		}),
		uboot.Uint("after_2", uboot.UintAfter, func(c *uboot.Context) error {
			// select {}
			c.Printf("after_2")
			return nil
		}),
		uboot.Uint("front", uboot.UintFront, func(c *uboot.Context) error {
			c.Printf("front")

			time.Sleep(500 * time.Millisecond)
			return nil
		}),
	).Start()
}
