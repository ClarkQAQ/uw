package umap

import (
	"sync"
)

type DefvKey interface {
	string | ~int | ~int64 | ~float64 | ~uint64 | uintptr | bool
}

type Defv[K DefvKey, V any] struct {
	pool   *sync.Pool
	loader func(K) (V, error)
}

func NewDefv[K DefvKey, V any](loader func(K) (V, error)) *Defv[K, V] {
	if loader == nil {
		loader = func(K) (V, error) {
			var v V
			return v, nil
		}
	}

	d := &Defv[K, V]{loader: loader}

	d.pool = &sync.Pool{
		New: func() interface{} {
			return &DefvSession[K, V]{
				d:  d,
				mu: &sync.RWMutex{},
				m:  make(map[K]V),
			}
		},
	}

	return d
}

type DefvSession[K DefvKey, V any] struct {
	d  *Defv[K, V]
	mu *sync.RWMutex
	m  map[K]V
}

func (d *Defv[K, V]) Session() *DefvSession[K, V] {
	return d.pool.Get().(*DefvSession[K, V])
}

func (s *DefvSession[K, V]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m = make(map[K]V)
	s.d.pool.Put(s)
}

func (s *DefvSession[K, V]) Load(key K) (V, error) {
	s.mu.RLock()
	if val, ok := s.m[key]; ok {
		s.mu.RUnlock()
		return val, nil
	}
	s.mu.RUnlock()

	val, e := s.d.loader(key)
	if e != nil {
		return val, e
	}

	s.mu.Lock()
	s.m[key] = val
	s.mu.Unlock()
	return val, nil
}

func (s *DefvSession[K, V]) Set(key K, val V) {
	s.mu.Lock()
	s.m[key] = val
	s.mu.Unlock()
}

func (s *DefvSession[K, V]) Change(key K, cb func(V) (V, error)) error {
	val, e := s.Load(key)
	if e != nil {
		return e
	}

	if val, e = cb(val); e != nil {
		return e
	}

	s.Set(key, val)
	return nil
}

func (s *DefvSession[K, V]) Delete(key K) {
	s.mu.Lock()
	delete(s.m, key)
	s.mu.Unlock()
}
