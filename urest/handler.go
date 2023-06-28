package urest

import (
	"fmt"
	"net/http"
	"reflect"
)

type (
	Handler                      func(c *Context)
	Recover                      func(c *Context, e error)
	RequestHandler[req, res any] func(c *Context, input *req) (res, error)
	Request[req any]             func(r *http.Request, input *req) error
	Response[res any]            struct {
		Response func(w http.ResponseWriter, resp res)
		Field    FieldFn
	}
	AnyRequest  func(r *http.Request, input any) error
	AnyResponse struct {
		Response func(w http.ResponseWriter, resp any)
		Field    FieldFn
	}
)

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
	Invoke(request AnyRequest, response AnyResponse, recover_ Recover, hf *HandlerField) (string, string, Handler, error)
}

// 方法结构
type Methodot[req, res any] struct {
	summary        string // 摘要
	detail         string // 描述
	customRequest  Request[req]
	customResponse Response[res]
	customRecover  Recover
	handler        RequestHandler[req, res]
}

// 新建方法
func Method[req, res any](handler RequestHandler[req, res]) *Methodot[req, res] {
	return &Methodot[req, res]{
		summary: "N/A",
		detail:  "N/A",
		handler: handler,
	}
}

func (m *Methodot[req, res]) Summary(summary string) *Methodot[req, res] {
	m.summary = summary
	return m
}

func (m *Methodot[req, res]) Detail(detail string) *Methodot[req, res] {
	m.detail = detail
	return m
}

func (m *Methodot[req, res]) Request(fn Request[req]) *Methodot[req, res] {
	m.customRequest = fn
	return m
}

func (m *Methodot[req, res]) Response(fn func(w http.ResponseWriter, resp res), rfn ...FieldFn) *Methodot[req, res] {
	m.customResponse.Response = fn
	if len(rfn) > 0 {
		m.customResponse.Field = rfn[0]
	}
	return m
}

func (m *Methodot[req, res]) Recover(fn Recover) *Methodot[req, res] {
	m.customRecover = fn
	return m
}

func (m *Methodot[req, res]) Handle(handler RequestHandler[req, res]) *Methodot[req, res] {
	m.handler = handler
	return m
}

func (m *Methodot[req, res]) Invoke(request AnyRequest, response AnyResponse, recover_ Recover, hf *HandlerField) (string, string, Handler, error) {
	if m.customRequest != nil {
		request = func(r *http.Request, input any) error {
			return m.customRequest(r, input.(*req))
		}
	} else if request == nil {
		request = func(r *http.Request, input any) error {
			return DefaultRequest(r, input.(*req))
		}
	}

	if m.customResponse.Response != nil {
		response.Response = func(w http.ResponseWriter, resp any) {
			m.customResponse.Response(w, resp.(res))
		}
	} else if response.Response == nil {
		response.Response = func(w http.ResponseWriter, resp any) {
			DefaultResponse(w, resp)
		}
	}

	if m.customResponse.Field != nil {
		response.Field = m.customResponse.Field
	}

	if m.customRecover != nil {
		recover_ = m.customRecover
	} else if recover_ == nil {
		recover_ = DefaultRecover
	}

	if v := reflect.TypeOf(new(req)).
		Elem(); v.Kind() == reflect.Struct {
		l, e := reflectField(false, v)
		if e != nil {
			return "", "", nil, fmt.Errorf("reflect request field error: %w", e)
		}

		hf.Request = append(hf.Request, l...)
	}

	l, e := reflectField(true, reflect.TypeOf(new(res)).
		Elem())
	if e != nil {
		return "", "", nil, fmt.Errorf("reflect response field error: %w", e)
	}

	if response.Field != nil {
		l = response.Field(l)
	}

	hf.Response = append(hf.Response, l...)

	return m.summary, m.detail, func(c *Context) {
		inp := new(req)
		if e := request(c.Req, inp); e != nil {
			recover_(c, e)
		}

		resp, e := m.handler(c, inp)
		if e != nil {
			recover_(c, e)
		}

		response.Response(c.Writer, resp)
	}, nil
}
