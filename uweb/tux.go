package uweb

import (
	"io"
	"net"
	"net/http"
	"sync"

	"uw/utree"
)

type Uweb struct {
	*Group                          // 路由组
	tree   *utree.Tree[HandlerList] // 路由树

	contextPool *sync.Pool
}

func New() *Uweb {
	uweb := &Uweb{}

	uweb.Group = &Group{uweb, "/", nil, nil}
	uweb.tree = utree.New[HandlerList]()
	uweb.contextPool = &sync.Pool{
		New: func() interface{} {
			return uweb.newContext()
		},
	}

	return uweb
}

func (uweb *Uweb) DumpRoute() []*utree.DumpValue[HandlerList] {
	return uweb.tree.Dump()
}

func defaultNotFound(c *Context) {
	c.Writer.WriteHeader(http.StatusNotFound)
	c.Writer.body.WriteString("404 NOT FOUND:" + c.Req.URL.Path)
}

func (uweb *Uweb) Handle(c *Context) {
	c.handlerList, c.vpath = uweb.tree.Get(c.Req.Method + "@" + c.Req.URL.Path)
	if len(c.handlerList) == 0 {
		c.handlerList = append(uweb.middleware, defaultNotFound)
	}

	c.Next()
}

func (uweb *Uweb) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := uweb.contextPool.Get().(*Context)
	defer func(t *Uweb, ctx *Context) {
		ctx.reset()
		t.contextPool.Put(c)
	}(uweb, c)

	c.use(w, req)

	uweb.Handle(c)

	if c.index < -10 {
		if c.index < -50 {
			panic(nil)
		}

		return
	}

	w.WriteHeader(c.Writer.status)

	if _, e := io.Copy(w, c.Writer.body); e != nil {
		panic(e)
	}
}

func (uweb *Uweb) ServeAddr(addr string) (*http.Server, error) {
	net, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, e
	}

	return uweb.ServeListener(net)
}

func (uweb *Uweb) ServeListener(l net.Listener) (*http.Server, error) {
	http := &http.Server{Handler: uweb}

	return http, http.Serve(l)
}
