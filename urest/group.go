package urest

import (
	"fmt"
	"net/http"
	"strings"
)

type Groupor interface {
	Invoke(r *Rest, prefix, tags []string, req AnyRequest,
		resp AnyResponse, recover_ Recover, handlerList []Handler) error
}

// 群组元数据
type Groupot struct {
	prefix         []string
	tags           []string
	customRequest  AnyRequest
	customResponse AnyResponse
	customRecover  Recover
	structs        []any
}

func Group(prefix string, structs ...any) *Groupot {
	return &Groupot{
		prefix:  strings.Split(prefix, "/"),
		structs: structs,
	}
}

func (g *Groupot) Request(fn AnyRequest) *Groupot {
	g.customRequest = fn
	return g
}

func (g *Groupot) Response(fn func(w http.ResponseWriter, resp any), rfn ...FieldFn) *Groupot {
	g.customResponse.Response = fn
	if len(rfn) > 0 {
		g.customResponse.Field = rfn[0]
	}
	return g
}

func (g *Groupot) Recover(fn Recover) *Groupot {
	g.customRecover = fn
	return g
}

func (g *Groupot) Tags(ss ...string) *Groupot {
	g.tags = append(g.tags, ss...)
	return g
}

func (g *Groupot) Invoke(r *Rest, prefix, tags []string, req AnyRequest,
	resp AnyResponse, recover_ Recover, handlerList []Handler,
) error {
	prefix, tags = append(prefix, g.prefix...), append(tags, g.tags...)

	if g.customRequest != nil {
		req = g.customRequest
	}
	if g.customResponse.Response != nil {
		resp.Response = g.customResponse.Response
	}
	if g.customResponse.Field != nil {
		resp.Field = g.customResponse.Field
	}
	if g.customRecover != nil {
		recover_ = g.customRecover
	}

	for i := 0; i < len(g.structs); i++ {
		srv, e := structReflectMethodValue(g.structs[i])
		if e != nil {
			return e
		}

		for i := 0; i < len(srv); i++ {
			if mw, ok := srv[i].Value.Interface().(func() Middlewareor); ok {
				handlerList = append(handlerList, mw().Invoke())
			}
		}

		for i := 0; i < len(srv); i++ {
			switch fn := srv[i].Value.Interface().(type) {
			case func() Groupor:
				if e := fn().Invoke(r, prefix, tags, req,
					resp, recover_, handlerList); e != nil {
					return e
				}
			case func() Methodor:
				rt := &Restot{
					method:       strings.ToUpper(srv[i].Name),
					path:         prefix,
					tags:         tags,
					handlerField: &HandlerField{},
					handlerList:  append(handlerList, func(_ *Context) {}),
				}

				rt.summary, rt.detail, rt.handlerList[len(rt.handlerList)-1], e = fn().
					Invoke(req, resp, recover_, rt.handlerField)
				if e != nil {
					return fmt.Errorf("urest: %s method %s.%s: %s",
						rt.method, srv[i].Type.Name(), srv[i].Name, e)
				}

				r.restotList = append(r.restotList, rt)
			}
		}
	}

	return nil
}
