package urest

import (
	"net/http"
	"sync"
)

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			m:            nil,
			Req:          nil,
			Writer:       nil,
			ctxStore:     nil,
			ctxStoreLock: new(sync.RWMutex),
			index:        -1,
		}
	},
}

type Context struct {
	m      *Restot
	Req    *http.Request
	Writer http.ResponseWriter

	ctxStore     map[string]interface{} // 上下文存储
	ctxStoreLock *sync.RWMutex          // 上下文存储锁

	index int // 当前执行的处理函数索引
}

func getContext() *Context {
	return contextPool.Get().(*Context)
}

func (c *Context) release() {
	c.Req = nil
	c.Writer = nil
	c.ctxStore = nil
	c.index = -1

	contextPool.Put(c)
}

func (c *Context) Set(key string, value interface{}) {
	c.ctxStoreLock.Lock()
	defer c.ctxStoreLock.Unlock()

	if c.ctxStore == nil {
		c.ctxStore = make(map[string]interface{})
	}

	c.ctxStore[key] = value
}

func (c *Context) Get(key string) (value interface{}, ok bool) {
	c.ctxStoreLock.RLock()
	defer c.ctxStoreLock.RUnlock()

	if c.ctxStore == nil {
		return nil, false
	}

	value, ok = c.ctxStore[key]
	return
}

func (c *Context) Delete(key string) {
	c.ctxStoreLock.Lock()
	defer c.ctxStoreLock.Unlock()

	if c.ctxStore == nil {
		return
	}

	delete(c.ctxStore, key)
}

func (c *Context) Index() (int, int) {
	return len(c.m.handlerList), c.index
}

func (c *Context) End() {
	c.index = -10
	panic(nil)
}

func (c *Context) Close() {
	c.index = -100
	panic(nil)
}

func (c *Context) Next() {
	for c.index++; c.index > -1 && c.index < len(c.m.handlerList); c.index++ {
		c.m.handlerList[c.index](c)
	}
}
