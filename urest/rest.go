package urest

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"uw/ulog"
	"uw/uweb"
)

var (
	ErrValueNotStructOrPointer = errors.New("value is not struct or pointer")
	ErrInvalidType             = errors.New("invalid type")
)

type Rest struct {
	restotList []*Restot
}

func NewRest(structs ...any) (*Rest, error) {
	r := &Rest{}
	return r, r.Rest(structs...)
}

func (r *Rest) Rest(structs ...any) error {
	mps := []*Restot{}

	for i := 0; i < len(structs); i++ {
		srv, e := structReflectMethodValue(structs[i])
		if e != nil {
			return e
		}

		for i := 0; i < len(srv); i++ {
			if g, ok := srv[i].Value.Interface().(func() Groupor); ok {
				if e := g().Invoke(r, []string{"/"}, nil, nil,
					AnyResponse{}, nil, nil); e != nil {
					return e
				}
			}
		}
	}

	r.restotList = append(r.restotList, mps...)
	return nil
}

func (r *Rest) Invoke() []*Restot {
	return r.restotList
}

func (r *Rest) BindUweb(u *uweb.Group) error {
	for i := 0; i < len(r.restotList); i++ {
		m := r.restotList[i]

		u.Method(m.Method(), m.Path(), func(c *uweb.Context) {
			c.ResponseWriter(func(w http.ResponseWriter) {
				m.ServeHTTP(w, c.Req)
			})
		})
	}

	return nil
}

type Restot struct {
	method       string
	path         []string
	tags         []string
	summary      string
	detail       string
	handlerField *HandlerField
	handlerList  []Handler
}

func (rt *Restot) Method() string {
	return rt.method
}

func (rt *Restot) Path() string {
	return path.Join(rt.path...)
}

func (rt *Restot) HandlerList() []Handler {
	return rt.handlerList
}

func (rt *Restot) Summary() string {
	return rt.summary
}

func (rt *Restot) Detail() string {
	return rt.detail
}

func (rt *Restot) HandlerField() *HandlerField {
	return rt.handlerField
}

func (rt *Restot) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := getContext()
	defer c.release()

	c.m = rt
	c.Req = r
	c.Writer = w

	defer func() {
		if r := recover(); r != nil {
			c.Writer.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(c.Writer, "InternalServerError: %v", r)
			ulog.Error("Internal Server Error: %v", r)
		} else if c.index == -100 {
			panic(nil)
		}
	}()

	c.Next()
}
