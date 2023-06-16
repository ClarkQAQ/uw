package uboot

import (
	"fmt"
	"time"
)

type UintType uint8

const (
	UintFront      UintType = iota // 最早运行 (运行时机: 1)
	UintBackground                 // 后台运行 (运行时机: 2)
	UintNormal                     // 默认运行 (运行时机: 3)
	UintDaemon                     // 守护运行 (运行时机: 4) (函数退出后, 会再次运行)
	UintAfter                      // 后续运行 (运行时机: 5)
)

type UintHandler func(c *Context) error

type UintAgent struct {
	name    string        // 名称
	handler UintHandler   // 处理函数
	utype   UintType      // 运行时机
	recover bool          // 错误恢复/无视
	timeout time.Duration // 超时时间
}

func UintTypeString(utype UintType) string {
	switch utype {
	case UintFront:
		return "front"
	case UintBackground:
		return "background"
	case UintNormal:
		return "normal"
	case UintDaemon:
		return "daemon"
	case UintAfter:
		return "after"
	default:
		return "unknown"
	}
}

func Uint(name string, utype UintType, handler UintHandler) *UintAgent {
	return &UintAgent{
		name:    name,
		handler: handler,
		utype:   utype,
		recover: false,
	}
}

func (u *UintAgent) Timeout(t time.Duration) *UintAgent {
	u.timeout = t
	return u
}

func (u *UintAgent) Recover() *UintAgent {
	u.recover = true
	return u
}

func (u *UintAgent) start(c *Context) {
	c.Printf("uint starting")

	defer func() {
		if r := recover(); r != nil {
			if !u.recover {
				c.Printf("uint start error: %v", r)
				panic(fmt.Sprintf("[%s] uint start panic: %v", u.name, r))
			}

			c.Printf("uint start error: %v", r)
		}

		if cwc := c.b.require.Get(u.name); cwc != nil {
			cwc.Cancel()
		}

		c.Cancel()
	}()

	c.timeout()
	defer c.cancelTimeout()

	if e := u.handler(c); e != nil {
		panic(e)
	}

	c.Printf("uint success")
}
