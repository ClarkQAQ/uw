package urest

import (
	"errors"

	"uw/uweb"
)

var (
	// ErrvalueNotStructOrPointer value is not struct or pointer / value 不是结构体或指针
	ErrValueNotStructOrPointer = errors.New("value is not struct or pointer")
	ErrInvalidType             = errors.New("invalid type")
)

type Methodop struct {
	Method      string
	Path        string
	Tags        []string
	Summary     string
	Description string

	Handler []Handler
}

func Invoke(gt any) ([]*Methodop, error) {
	rvs, e := structReflectValue(gt)
	if e != nil {
		return nil, e
	}

	mps := []*Methodop{}
	for i := 0; i < len(rvs); i++ {
		if group, ok := rvs[i].Value.Interface().(func() Groupor); ok {
			tmps, e := group().Invoke("/", nil, nil)
			if e != nil {
				return nil, e
			}

			mps = append(mps, tmps...)
		}
	}

	return mps, nil
}

func BindUweb(g *uweb.Group, val any) error {
	return nil
}
