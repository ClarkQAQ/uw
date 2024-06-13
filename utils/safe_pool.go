package utils

import "sync"

type SafePool[T any] struct {
	p *sync.Pool
}

func NewSafePool[T any](new func() T) *SafePool[T] {
	return &SafePool[T]{&sync.Pool{New: func() any {
		return new()
	}}}
}

func (sp SafePool[T]) Get() T {
	return sp.p.Get().(T)
}

func (sp SafePool[T]) Put(v T) {
	sp.p.Put(v)
}
