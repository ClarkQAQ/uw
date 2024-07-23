package uweb

import (
	"path"
	"reflect"
	"strings"
)

// 路由组
type Group struct {
	uweb       *Uweb         // Uweb
	prefix     string        // 组前缀
	parent     *Group        // 上级组
	middleware []HandlerFunc // 组定义中间件
}

func (g *Group) NewGroup(prefix string) *Group {
	return &Group{
		uweb:   g.uweb,
		prefix: path.Join(g.prefix, prefix),
		parent: g,
	}
}

func (g *Group) generateHandlerList(handler ...HandlerFunc) []HandlerFunc {
	handles := make([]HandlerFunc, 0, len(g.middleware)+len(handler))

	for {
		handles = append(handles, g.middleware...)

		if g.parent != nil {
			g = g.parent
			continue
		}

		break
	}

	handles = append(handles, handler...)
	return handles
}

func (g *Group) GroupTx(prefix string, f func(g *Group)) {
	f(g.NewGroup(prefix))
}

func (g *Group) Use(middleware ...HandlerFunc) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *Group) Method(method, part string, handler HandlerFunc) {
	g.uweb.tree.Set(strings.ToUpper(method)+"@"+path.Join(g.prefix, part),
		g.generateHandlerList(handler))
}

func (g *Group) Get(part string, handler HandlerFunc) {
	g.Method("GET", part, handler)
}

func (g *Group) Post(part string, handler HandlerFunc) {
	g.Method("POST", part, handler)
}

func (g *Group) Put(part string, handler HandlerFunc) {
	g.Method("PUT", part, handler)
}

func (g *Group) Delete(part string, handler HandlerFunc) {
	g.Method("DELETE", part, handler)
}

func (g *Group) Patch(part string, handler HandlerFunc) {
	g.Method("PATCH", part, handler)
}

func (g *Group) Head(part string, handler HandlerFunc) {
	g.Method("HEAD", part, handler)
}

func (g *Group) Options(part string, handler HandlerFunc) {
	g.Method("OPTIONS", part, handler)
}

func (g *Group) Any(part string, handler HandlerFunc) {
	g.Split("GET,POST,PUT,DELETE,PATCH,HEAD,OPTIONS", part, handler)
}

func (g *Group) Split(methods, part string, handler HandlerFunc) {
	for _, method := range strings.Split(methods, ",") {
		g.Method(strings.TrimSpace(method), part, handler)
	}
}

func (g *Group) Object(part string, object interface{}) {
	v := reflect.ValueOf(object)
	t := v.Type()

	if v.Kind() == reflect.Struct {
		newValue := reflect.New(t)
		newValue.Elem().Set(v)
		v = newValue
		t = v.Type()
	}

	g.GroupTx(part, func(r *Group) {
		for i := 0; i < v.NumMethod(); i++ {
			methodName := strings.ToUpper(t.Method(i).Name)

			if strings.HasPrefix(methodName, "MIDDLEWARE") {
				if h, ok := v.Method(i).Interface().(func(*Context)); ok {
					r.Use(h)
				}
				continue
			}

			if h, ok := v.Method(i).Interface().(func(*Context)); ok {
				r.Method(methodName, "/", h)
			}
		}
	})
}
