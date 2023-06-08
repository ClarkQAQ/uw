package urest

import (
	"path"
	"strings"
)

type Groupor interface {
	Invoke(prefix string, tags []string, handlerList []Handler) ([]*Methodop, error)
}

// 群组元数据
type Groupot struct {
	prefix  string
	tags    []string
	structs []any
}

func Group(prefix string, structs ...any) *Groupot {
	return &Groupot{
		prefix:  prefix,
		structs: structs,
	}
}

func (g *Groupot) Tags(ss ...string) *Groupot {
	g.tags = append(g.tags, ss...)
	return g
}

func (g *Groupot) Invoke(prefix string, tags []string, handlerList []Handler) ([]*Methodop, error) {
	mps := []*Methodop{}
	prefix = path.Join(prefix, g.prefix)
	tags = append(tags, g.tags...)

	for i := 0; i < len(g.structs); i++ {
		rvs, e := structReflectValue(g.structs[i])
		if e != nil {
			return nil, e
		}

		for i := 0; i < len(rvs); i++ {
			if mw, ok := rvs[i].Value.Interface().(func() Middlewareor); ok {
				handlerList = append(handlerList, mw().Invoke())
			}
		}

		for i := 0; i < len(rvs); i++ {
			switch val := rvs[i].Value.Interface().(type) {
			case func() Groupor:
				tmps, e := val().Invoke(prefix, tags, handlerList)
				if e != nil {
					return nil, e
				}

				mps = append(mps, tmps...)
			case func() Methodor:
				mp, e := val().Invoke(strings.ToUpper(rvs[i].Method.Name),
					prefix, g.tags, handlerList)
				if e != nil {
					return nil, e
				}

				mps = append(mps, mp)
			}
		}
	}

	return mps, nil
}
