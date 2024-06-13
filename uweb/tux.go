package uweb

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"uw/utils"
	"uw/utree"
)

type Uweb struct {
	*Group                                // router group, this is the root group
	tree        *utree.Tree[HandlerList]  // router tree, use utree
	contextPool *utils.SafePool[*Context] // context pool
}

func New() *Uweb {
	uweb := &Uweb{}

	uweb.Group = &Group{uweb, "/", nil, nil}
	uweb.tree = utree.New[HandlerList]()
	uweb.contextPool = utils.NewSafePool(func() *Context {
		return uweb.newContext()
	})

	return uweb
}

func (uweb *Uweb) DumpRoute() []*utree.DumpValue[HandlerList] {
	return uweb.tree.Dump()
}

func defaultNotFound(c *Context) {
	http.Error(c.Writer, "404 NOT FOUND:"+c.Req.URL.Path, http.StatusNotFound)
}

func (uweb *Uweb) Handle(c *Context) {
	c.handlerList, c.vpath = uweb.tree.Get(c.Req.Method + "@" + c.Req.URL.Path)
	if len(c.handlerList) == 0 {
		c.handlerList = append(uweb.middleware, defaultNotFound)
	}

	defer func() {
		if r := recover(); r != nil && c.index > -1 {
			c.Clean()
			http.Error(c.Writer,
				fmt.Sprintf("%s, Recover: %v",
					http.StatusText(http.StatusInternalServerError), r),
				http.StatusInternalServerError)
		}
	}()

	c.Next()
}

func (uweb *Uweb) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := uweb.contextPool.Get()
	defer func(t *Uweb, ctx *Context) {
		ctx.reset()
		t.contextPool.Put(c)
	}(uweb, c)

	c.use(w, req)

	uweb.Handle(c)

	if c.index < EndIndex {
		if c.index <= CloseIndex {
			panic(http.ErrAbortHandler)
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
