package uboot

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"uw/ulog"
	"uw/umap"
)

const ubootLogo = `
█    ██  ▄▄▄▄    ▒█████   ▒█████  ▄▄▄█████▓
██  ▓██▒▓█████▄ ▒██▒  ██▒▒██▒  ██▒▓  ██▒ ▓▒
▓██  ▒██░▒██▒ ▄██▒██░  ██▒▒██░  ██▒▒ ▓██░ ▒░
▓▓█  ░██░▒██░█▀  ▒██   ██░▒██   ██░░ ▓██▓ ░ 
▒▒█████▓ ░▓█  ▀█▓░ ████▓▒░░ ████▓▒░  ▒██▒ ░ 
░▒▓▒ ▒ ▒ ░▒▓███▀▒░ ▒░▒░▒░ ░ ▒░▒░▒░   ▒ ░░   
░░▒░ ░ ░ ▒░▒   ░   ░ ▒ ▒░   ░ ▒ ▒░     ░    
░░░ ░ ░  ░    ░ ░ ░ ░ ▒  ░ ░ ░ ▒    ░      
  ░      ░          ░ ░      ░ ░                													   
`

var defaultBootTimeout = 60 * time.Second

type Printf func(format string, args ...interface{})

type ContextWithCancel struct {
	context.Context
	Cancel context.CancelFunc
}

func newContextWithCancel() *ContextWithCancel {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContextWithCancel{
		Context: ctx,
		Cancel:  cancel,
	}
}

type Boot struct {
	bootTimeout    time.Duration // 启动超时时间
	printf         Printf        // 打印函数
	frontUint      []*UintAgent  // 预启动模块
	backgroundUint []*UintAgent  // 后台模块
	normalUint     []*UintAgent  // 默认模块
	afterUint      []*UintAgent  // 延后启动模块
	lock           *sync.Mutex   // 用于锁定 Boot 对象的 Start，防止重复启动, 类似 once 的作用
	storage        *umap.Hmap[string, interface{}]
	require        *umap.Hmap[string, *ContextWithCancel] // 用于模块间的依赖

	allowNameRepeat bool // 允许模块名重复
}

func NewBoot() *Boot {
	b := &Boot{
		bootTimeout:    defaultBootTimeout,
		frontUint:      []*UintAgent{},
		backgroundUint: []*UintAgent{},
		normalUint:     []*UintAgent{},
		afterUint:      []*UintAgent{},
		lock:           &sync.Mutex{},
		storage:        umap.NewHmap[string, interface{}](),
		require:        umap.NewHmap[string, *ContextWithCancel](),
	}

	b.SetPrintf(ulog.Printf)
	return b
}

func (b *Boot) SetPrintf(l Printf) *Boot {
	b.printf = func(format string, args ...interface{}) {
		l(ulog.SetANSI(ulog.ANSI.Bold, "[UBOOT]") + " " + fmt.Sprintf(format, args...))
	}

	return b
}

func (b *Boot) BootTimeout(t time.Duration) *Boot {
	b.bootTimeout = t
	return b
}

func (b *Boot) AllowNameRepeat() *Boot {
	b.allowNameRepeat = true
	return b
}

func (b *Boot) Register(uintAgents ...*UintAgent) *Boot {
	for i := 0; i < len(uintAgents); i++ {
		switch uintAgents[i].utype {
		case UintFront:
			b.frontUint = append(b.frontUint, uintAgents[i])
		case UintBackground:
			b.backgroundUint = append(b.backgroundUint, uintAgents[i])
		case UintNormal:
			b.normalUint = append(b.normalUint, uintAgents[i])
		case UintAfter:
			b.afterUint = append(b.afterUint, uintAgents[i])
		}

		if b.require.Get(uintAgents[i].name) != nil && !b.allowNameRepeat {
			b.printf("register uint name repeat: %s", uintAgents[i].name)
			panic("uboot: register uint name repeat: " + uintAgents[i].name)
		}

		b.require.Set(uintAgents[i].name, newContextWithCancel())
	}

	return b
}

func (b *Boot) newContext(u *UintAgent) *Context {
	prefix := ulog.ANSI.Bold + "[" + strings.ToUpper(UintTypeString(u.utype)) +
		":" + u.name + "]" + ulog.ANSI.Reset + " "

	c := &Context{
		b: b,
		u: u,
		printf: func(format string, args ...interface{}) {
			b.printf(prefix + ulog.ANSI.Grey + fmt.Sprintf(format, args...) + ulog.ANSI.Reset)
		},
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	return c
}

func (b *Boot) Start() bool {
	if !b.lock.TryLock() {
		return false
	}

	os.Stdout.WriteString(ubootLogo)
	b.printf(ulog.SetANSI(ulog.ANSI.Bold, "uboot start"))
	if b.bootTimeout > 0 {
		t := time.AfterFunc(b.bootTimeout, func() {
			b.printf(ulog.SetANSI(ulog.ANSI.Magenta, "normal uint start timeout!"))
			panic("normal uint start timeout!")
		})

		defer t.Stop()
	}

	defer func() {
		b.printf(ulog.SetANSI(ulog.ANSI.Bold, "uboot done"))
	}()

	if len(b.frontUint) > 0 {
		b.printf(ulog.SetANSI(ulog.ANSI.Cyan, "start front uint"))
		for i := 0; i < len(b.frontUint); i++ {
			b.frontUint[i].start(b.newContext(b.frontUint[i]))
		}
		b.printf(ulog.SetANSI(ulog.ANSI.Green, "start front uint done"))
	}

	if len(b.backgroundUint) > 0 {
		wg := &sync.WaitGroup{}
		defer func() {
			b.printf(ulog.SetANSI(ulog.ANSI.Green, "waiting for all background uint done"))
			wg.Wait()
			b.printf(ulog.SetANSI(ulog.ANSI.Green, "all background uint done"))
		}()

		b.printf(ulog.SetANSI(ulog.ANSI.Cyan, "create background uint"))
		for i := 0; i < len(b.backgroundUint); i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				b.backgroundUint[i].start(b.newContext(b.backgroundUint[i]))
			}(i)
		}
		b.printf(ulog.SetANSI(ulog.ANSI.Green, "create background uint done"))
	}

	if len(b.normalUint) > 0 {
		wg := &sync.WaitGroup{}

		b.printf(ulog.SetANSI(ulog.ANSI.Cyan, "create normal uint"))
		for i := 0; i < len(b.normalUint); i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				b.normalUint[i].start(b.newContext(b.normalUint[i]))
			}(i)
		}
		b.printf(ulog.SetANSI(ulog.ANSI.Green, "create normal uint done"))
		b.printf(ulog.SetANSI(ulog.ANSI.Blue, "waiting for all normal uint done"))
		wg.Wait()
		b.printf(ulog.SetANSI(ulog.ANSI.Green, "all normal uint done"))
	}

	if len(b.afterUint) > 0 {
		b.printf(ulog.SetANSI(ulog.ANSI.Cyan, "start after uint"))
		for i := 0; i < len(b.afterUint); i++ {
			b.afterUint[i].start(b.newContext(b.afterUint[i]))
		}
		b.printf(ulog.SetANSI(ulog.ANSI.Green, "start after uint done"))
	}

	return true
}
