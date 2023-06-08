package urest

import (
	"context"
	"fmt"
	"net/http"
)

type Handler func(c *Context)

type Middlewareor interface {
	Invoke() Handler
}

type Middlewareot struct {
	handler Handler
}

func (m *Middlewareot) Invoke() Handler {
	return m.handler
}

// 中间件接口
func Middleware(handler Handler) Middlewareor {
	return &Middlewareot{
		handler: handler,
	}
}

// 方法接口
type Methodor interface {
	Invoke(method, prefix string, tags []string, handler []Handler) (*Methodop, error)
}

// 方法结构
type Methodot[i, o any] struct {
	handler     func(ctx context.Context, input *i) (o, error)
	tags        []string // 标签
	summary     string   // 摘要
	description string   // 描述
}

// 新建方法
func Method[req, res any](handler func(ctx context.Context, input *req) (res, error)) *Methodot[req, res] {
	return &Methodot[req, res]{
		summary:     "N/A",
		description: "N/A",
		handler:     handler,
	}
}

func (m *Methodot[i, o]) Tags(ss ...string) *Methodot[i, o] {
	m.tags = append(m.tags, ss...)
	return m
}

func (m *Methodot[i, o]) Summary(summary string) *Methodot[i, o] {
	m.summary = summary
	return m
}

func (m *Methodot[i, o]) Description(desc string) *Methodot[i, o] {
	m.description = desc
	return m
}

func (m *Methodot[i, o]) Handler(handler func(ctx context.Context, input *i) (o, error)) *Methodot[i, o] {
	m.handler = handler
	return m
}

func (m *Methodot[req, res]) Invoke(method, prefix string, tags []string, handler []Handler) (*Methodop, error) {
	mp := &Methodop{
		Method:      method,
		Path:        prefix,
		Tags:        append(tags, m.tags...),
		Summary:     m.summary,
		Description: m.description,
	}

	h, e := m.getHandler()
	if e != nil {
		return nil, e
	}

	mp.Handler = append(handler, h)
	return mp, nil
}

func (m *Methodot[i, o]) getHandler() (Handler, error) {
	return func(c *Context) {
		// ctx := r.Context()

		// inp := new(req)
		// if e := parseRequest(r, inp); e != nil {
		// 	writeError(w, e)
		// 	return
		// }

		// res, e := m.handler(ctx, inp)
		// if e != nil {
		// 	writeError(w, e)
		// 	return
		// }

		// writeResponse(w, res)
	}, nil
}

func (m *Methodot[req, res]) Request(ctx context.Context, input interface{}) (interface{}, error) {
	inp, ok := input.(*req)
	if !ok {
		return nil, fmt.Errorf("%w of input: %T, expected: %T", ErrInvalidType, input, new(req))
	}

	return m.handler(ctx, inp)
}

func parseRequest(r *http.Request, inp interface{}) error {
	return nil
}

func writeError(w http.ResponseWriter, e error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "InternalServerError: %s", e.Error())
}

func writeResponse(w http.ResponseWriter, res interface{}) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%v", res)))
}
