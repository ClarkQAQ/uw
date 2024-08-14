package uboot

import (
	"context"
	"fmt"
	"time"
)

type Context struct {
	b            *Boot
	u            *UintAgent
	printf       Printf
	ctx          context.Context
	cancel       context.CancelFunc
	timeoutTimer *time.Timer
}

func (c *Context) Name() string {
	return c.u.name
}

func (c *Context) Recover() bool {
	return c.u.recover
}

func (c *Context) Timeout() time.Duration {
	return c.u.timeout
}

func (c *Context) Printf(format string, args ...interface{}) {
	c.printf(format, args...)
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) Cancel() {
	c.cancel()
}

func (c *Context) Require(ctx context.Context, name string) error {
	c.printf("require: %s", name)

	if cwc := c.b.require.Get(name); cwc != nil {
		c.cancelTimeout()
		defer c.timeout()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-cwc.Done():
			return nil
		}
	}

	return fmt.Errorf("require %s not found", name)
}

func (c *Context) timeout() {
	if c.u.timeout > 0 {
		if c.timeoutTimer != nil {
			c.timeoutTimer.Reset(c.u.timeout)
			return
		}

		c.timeoutTimer = time.AfterFunc(c.u.timeout, func() {
			c.Cancel()

			if !c.u.recover {
				c.Printf("uint start timeout")
				panic(fmt.Sprintf("[%s] uint start timeout", c.u.name))
			}

			c.Printf("uint start timeout")
		})
	}
}

func (c *Context) cancelTimeout() {
	if c.timeoutTimer != nil {
		c.timeoutTimer.Stop()
	}
}
